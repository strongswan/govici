// Copyright (C) 2025 Nick Rosbrook
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package vici

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"iter"
	"net"
	"slices"
	"sync"
	"time"
)

var (
	// Received unexpected response from server
	errUnexpectedResponse = errors.New("vici: unexpected response type")

	// Received EVENT_UNKNOWN from server
	errEventUnknown = errors.New("vici: unknown event type")
)

type clientConn struct {
	sync.Mutex
	conn net.Conn

	// Propagate errors from the listen() loop.
	err chan error

	// Read and write sequence counters used to associate command responses with
	// the caller. This keeps coherency when callers abandon their response, e.g.
	// because the context is cancelled or the deadline is exceeded.
	rseq uint64
	wseq uint64

	// Packet chan buffer. The listen() function is responseible for reading
	// all data from the server, and this chan buffer is used to dispatch
	// reponses to waiting callers.
	pc chan *Message

	events struct {
		sync.Mutex

		list        []string
		streaming   string
		subscribers map[chan<- Event]struct{}
	}
}

func newClientConn(conn net.Conn) *clientConn {
	cc := &clientConn{
		conn: conn,
		pc:   make(chan *Message, 128),
		err:  make(chan error, 1),
		events: struct {
			sync.Mutex
			list        []string
			streaming   string
			subscribers map[chan<- Event]struct{}
		}{
			list:        make([]string, 0),
			subscribers: make(map[chan<- Event]struct{}),
		},
	}

	return cc
}

func (cc *clientConn) Close() error {
	return cc.conn.Close()
}

// listen is responsible for reading all data from the server. It dispatches
// read packets depending on the type.
//
// For command and event registration responses, the packet is written to
// the packet chan buffer, which is then consumed by the original caller.
//
// For event packets, all subscribers are notified of the packet. For some event
// types, this may only be an internal subscriber associated with a streaming
// command.
func (cc *clientConn) listen() {
	defer cc.stop()

	for {
		p, err := cc.read()
		if err != nil {
			cc.err <- err
			return
		}

		switch p.header.ptype {
		case /* We received an event from the server. */
			pktEvent:

			e := Event{
				Name:      p.header.name,
				Message:   p,
				Timestamp: time.Now(),
			}
			cc.dispatch(e)

		case /* Responses to normal command requests and event registration. */
			pktCmdResponse,
			pktCmdUnkown,
			pktEventConfirm,
			pktEventUnknown:

			// Only increment this counter for direct response packets.
			cc.rseq++
			p.header.seq = cc.rseq

			select {
			case cc.pc <- p:
			default:
				// For now, silently drop the packet. Do not block
				// if the chan is full.
			}

		case /* These are only handled server-side, ignore. */
			pktCmdRequest,
			pktEventRegister,
			pktEventUnregister:

			continue
		default:
			/* We should not be here, unless a bogus message was received. */
			continue
		}
	}
}

func (cc *clientConn) stop() {
	close(cc.pc)

	cc.events.Lock()
	defer cc.events.Unlock()

	for c := range cc.events.subscribers {
		close(c)
	}
}

func (cc *clientConn) read() (*Message, error) {
	p := NewMessage()

	buf := make([]byte, 4 /* header length */)
	_, err := io.ReadFull(cc.conn, buf)
	if err != nil {
		return nil, err
	}
	pl := binary.BigEndian.Uint32(buf)

	buf = make([]byte, int(pl))
	_, err = io.ReadFull(cc.conn, buf)
	if err != nil {
		return nil, err
	}

	if err := p.decode(buf); err != nil {
		return nil, err
	}

	return p, nil
}

func (cc *clientConn) write(ctx context.Context, p *Message) error {
	if ctx == nil {
		return errors.New("context cannot be nil")
	}

	if p == nil {
		return errors.New("message cannot be nil")
	}

	if !p.packetIsValid() {
		return errors.New("packet header is invalid")
	}

	if !p.packetIsRequest() {
		return fmt.Errorf("invalid request with packet type %v", p.header.ptype)
	}

	buf := bytes.NewBuffer([]byte{})

	b, err := p.encode()
	if err != nil {
		return err
	}

	// Write the packet length
	if err := safePutUint32(buf, len(b)); err != nil {
		return err
	}

	// Write the payload
	_, err = buf.Write(b)
	if err != nil {
		return err
	}

	// Reset the write deadline in case a previous write was cancelled.
	if err := cc.conn.SetWriteDeadline(time.Time{}); err != nil {
		return err
	}

	rc := make(chan error, 1)
	go func() {
		defer close(rc)

		_, err = cc.conn.Write(buf.Bytes())
		rc <- err
	}()

	select {
	case <-ctx.Done():
		// User context was canceled or deadline exceeded. End the
		// write and bail. Once we set the deadline, wait for
		// the goroutune above to return.
		if err := cc.conn.SetWriteDeadline(time.Now()); err != nil {
			return err
		}

		if err := <-rc; err != nil {
			// Assuming the write did fail, return the context's
			// error for clarity.
			return ctx.Err()
		}
	case err := <-rc:
		if err != nil {
			return err
		}
	}

	// Increment the write sequence on successful writes.
	cc.wseq++

	return nil
}

func (cc *clientConn) wait(ctx context.Context) (*Message, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	// nolint
	defer cc.conn.SetReadDeadline(time.Time{})
	for {
		// Wait for the response packet as long as the context is not
		// cancelled, and as long as we continue receiving packets from
		// the server.
		deadline := time.Now().Add(5 * time.Second)

		if d, ok := ctx.Deadline(); ok && d.After(deadline) {
			deadline = d.Add(5 * time.Second)
		}

		if err := cc.conn.SetReadDeadline(deadline); err != nil {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case p, ok := <-cc.pc:
			if !ok {
				return nil, fmt.Errorf("vici: error waiting for response: %w", io.ErrClosedPipe)
			}

			if p.header.ptype != pktEvent {
				if p.header.seq != cc.wseq {
					// This is not the packet you're looking for...
					continue
				}
			}

			return p, nil

		case err, ok := <-cc.err:
			if !ok {
				err = io.ErrClosedPipe
			}

			return nil, fmt.Errorf("vici: error waiting for data: %w", err)
		}
	}
}

func (cc *clientConn) request(ctx context.Context, ptype uint8, name string, in *Message) (*Message, error) {
	if in == nil {
		in = NewMessage()
	}

	in.header = &header{
		ptype: ptype,
		name:  name,
	}

	if err := cc.write(ctx, in); err != nil {
		return nil, err
	}

	p, err := cc.wait(ctx)
	if err != nil {
		return nil, err
	}

	// Handle the response type according to the request type.
	switch ptype {
	case /* Command request */
		pktCmdRequest:

		if p.header.ptype != pktCmdResponse {
			return nil, fmt.Errorf("%w: %v", errUnexpectedResponse, p.header.ptype)
		}

	case /* Event (un)registration */
		pktEventRegister,
		pktEventUnregister:

		if p.header.ptype == pktEventUnknown {
			return nil, fmt.Errorf("%w: %v", errEventUnknown, name)
		}
		if p.header.ptype != pktEventConfirm {
			return nil, fmt.Errorf("%w: %v", errUnexpectedResponse, p.header.ptype)
		}
	default:
		// This should never happen.
		return nil, fmt.Errorf("internal error: invalid packet type %v", ptype)
	}

	return p, p.Err()
}

func (cc *clientConn) stream(ctx context.Context, cmd string, event string, in *Message) iter.Seq2[*Message, error] {
	return func(yield func(*Message, error) bool) {
		if in == nil {
			in = NewMessage()
		}

		in.header = &header{
			ptype: pktCmdRequest,
			name:  cmd,
		}

		// Initialize the associated event streaming.
		if _, err := cc.request(ctx, pktEventRegister, event, nil); err != nil {
			yield(nil, err)
			return
		}
		cc.events.Lock()
		cc.events.streaming = event
		cc.events.Unlock()
		defer func() {
			// nolint
			_, _ = cc.request(ctx, pktEventUnregister, event, nil)

			cc.events.Lock()
			cc.events.streaming = ""
			cc.events.Unlock()
		}()

		if err := cc.write(ctx, in); err != nil {
			yield(nil, err)
			return
		}

		for {
			p, err := cc.wait(ctx)
			if err != nil {
				yield(nil, err)
				return
			}

			switch p.header.ptype {
			case /* Event packet. There may be more. */
				pktEvent:

				if p.header.name != event {
					continue
				}

				if !yield(p, p.Err()) {
					return
				}
			case /* Command response, stream is complete. */
				pktCmdResponse:

				// If the last message contains an error,
				// propagate it. Otherwise, the previous event
				// should be the last message seen by the
				// caller.
				if err := p.Err(); err != nil {
					yield(p, p.Err())
				}

				return
			default:
				yield(nil, fmt.Errorf("%w: %v", errUnexpectedResponse, p.header.ptype))
				return
			}
		}
	}
}

func (cc *clientConn) call(ctx context.Context, cmd string, in *Message) (*Message, error) {
	return cc.request(ctx, pktCmdRequest, cmd, in)
}

func (cc *clientConn) subscribe(ctx context.Context, events ...string) error {
	cc.events.Lock()
	defer cc.events.Unlock()

	if events == nil {
		return errors.New("must specify at least one event")
	}

	for _, event := range events {
		if slices.Contains(cc.events.list, event) {
			continue
		}

		if _, err := cc.request(ctx, pktEventRegister, event, nil); err != nil {
			return err
		}

		cc.events.list = append(cc.events.list, event)
	}

	return nil
}

func (cc *clientConn) unsubscribe(ctx context.Context, events ...string) error {
	cc.events.Lock()
	defer cc.events.Unlock()

	if events == nil {
		events = slices.Clone(cc.events.list)
	}

	for _, event := range events {
		index := slices.Index(cc.events.list, event)
		if index < 0 {
			continue
		}

		if _, err := cc.request(ctx, pktEventUnregister, event, nil); err != nil {
			return err
		}

		cc.events.list = slices.Delete(cc.events.list, index, index+1)
	}

	return nil
}

func (cc *clientConn) notify(c chan<- Event) {
	cc.events.Lock()
	defer cc.events.Unlock()

	cc.events.subscribers[c] = struct{}{}
}

func (cc *clientConn) unnotify(c chan<- Event) {
	cc.events.Lock()
	defer cc.events.Unlock()

	delete(cc.events.subscribers, c)
}

func (cc *clientConn) dispatch(ev Event) {
	cc.events.Lock()
	defer cc.events.Unlock()

	if ev.Name == cc.events.streaming {
		// This event is associated with an active streaming call.
		// Dispatch internally only.
		select {
		case cc.pc <- ev.Message:
		default:
		}

		return
	}

	if !slices.Contains(cc.events.list, ev.Name) {
		// Nothing subscribed to this, ignore.
		return
	}

	for c := range cc.events.subscribers {
		select {
		case c <- ev:
		default:
		}
	}
}
