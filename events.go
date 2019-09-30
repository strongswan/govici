// Copyright (C) 2019 Nick Rosbrook
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
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

var (
	// Event listener channel was closed
	errChannelClosed = errors.New("vici: event listener channel closed")
)

type eventListener struct {
	*transport

	// Context supplied by caller through Listen.
	lctx context.Context

	ec     chan event
	events []string

	// Closing context and its cancel func are used to
	// indicate that this event listener has been Close()'d.
	//
	// Use a sync.Once to make sure that destroy() is a no-op
	// after first call to it.
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

type event struct {
	msg *Message
	err error
}

func newEventListener(ctx context.Context, t *transport) *eventListener {
	cctx, ccancel := context.WithCancel(context.Background())

	return &eventListener{
		transport: t,
		lctx:      ctx,
		ec:        make(chan event, 16),
		ctx:       cctx,
		cancel:    ccancel,
	}
}

// Close closes the event channel.
func (el *eventListener) Close() error {
	el.cancel()

	return nil
}

// Done returns if the event listener is closed.
func (el *eventListener) Done() <-chan struct{} {
	return el.ctx.Done()
}

func (el *eventListener) destroy() {
	el.once.Do(func() {
		el.unregisterEvents(el.events)
		close(el.ec)
		el.conn.Close()
	})
}

func (el *eventListener) listen(events []string) (err error) {
	err = el.registerEvents(events)
	if err != nil {
		return err
	}
	el.events = events

	go func() {
		defer el.destroy()

		for {
			select {
			case <-el.lctx.Done():
				// Caller cancelled their context.
				return

			case <-el.ctx.Done():
				// Closer context was cancelled.
				return

			default:
				var e event

				// Set a read deadline so that this loop can continue
				// at a reasonable pace. If the error is a timeout,
				// do not send it on the event channel.
				_ = el.conn.SetReadDeadline(time.Now().Add(time.Second))

				p, err := el.recv()
				if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						continue
					}
					e.err = err

					// Send error event and continue in loop.
					el.ec <- e
					continue
				}

				if p.ptype == pktEvent {
					e.msg = p.msg
					el.ec <- e
				}
			}
		}
	}()

	return nil
}

func (el *eventListener) nextEvent() (*Message, error) {
	e := <-el.ec
	if e.msg == nil && e.err == nil {
		return nil, errChannelClosed
	}

	return e.msg, e.err
}

func (el *eventListener) registerEvents(events []string) error {
	for i, e := range events {
		err := el.eventRegisterUnregister(e, true)
		if err != nil {
			el.unregisterEvents(events[:i])

			return err
		}
	}

	return nil
}

func (el *eventListener) unregisterEvents(events []string) {
	for _, e := range events {
		// nolint
		el.eventRegisterUnregister(e, false)
	}
}

func (el *eventListener) eventRegisterUnregister(event string, register bool) error {
	ptype := pktEventRegister
	if !register {
		ptype = pktEventUnregister
	}

	p, err := el.eventTransportCommunicate(newPacket(ptype, event, nil))
	if err != nil {
		return err
	}

	if p.ptype == pktEventUnknown {
		return fmt.Errorf("%v: %v", errEventUnknown, event)
	}

	if p.ptype != pktEventConfirm {
		return fmt.Errorf("%v:%v", errUnexpectedResponse, p.ptype)
	}

	return nil
}

func (el *eventListener) eventTransportCommunicate(pkt *packet) (*packet, error) {
	err := el.send(pkt)
	if err != nil {
		return nil, err
	}

	// Refresh deadline for communication.
	err = el.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return nil, err
	}

	p, err := el.recv()
	if err != nil {
		return nil, err
	}

	return p, nil
}
