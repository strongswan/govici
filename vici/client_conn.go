// Copyright (C) 2023 Nick Rosbrook
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
	"net"
)

const (
	headerLength = 4
)

var (
	// Received unexpected response from server
	errUnexpectedResponse = errors.New("vici: unexpected response type")

	// Received EVENT_UNKNOWN from server
	errEventUnknown = errors.New("vici: unknown event type")
)

type clientConn struct {
	network string
	addr    string
	dialer  func(ctx context.Context, network, addr string) (net.Conn, error)

	closed bool
	conn   net.Conn
}

func (cc *clientConn) dial(ctx context.Context) error {
	if !cc.closed && cc.conn != nil {
		return nil
	}

	conn, err := cc.dialer(ctx, cc.network, cc.addr)
	if err != nil {
		return err
	}

	cc.conn = conn
	cc.closed = false

	return nil
}

func (cc *clientConn) Close() error {
	if cc.closed || cc.conn == nil {
		return nil
	}

	cc.closed = true

	return cc.conn.Close()
}

func (cc *clientConn) packetWrite(ctx context.Context, m *Message) error {
	if err := cc.dial(ctx); err != nil {
		return err
	}

	rc := cc.asyncPacketWrite(m)
	select {
	case <-ctx.Done():
		// Disconnect on context deadline to avoid data ordering
		// problems with subsequent read/writes. Re-establish the
		// connection later.
		cc.Close()
		<-rc

		return ctx.Err()
	case err := <-rc:
		if err != nil {
			return err
		}
		return nil
	}
}

func (cc *clientConn) packetRead(ctx context.Context) (*Message, error) {
	if err := cc.dial(ctx); err != nil {
		return nil, err
	}

	rc := cc.asyncPacketRead()
	select {
	case <-ctx.Done():
		// Disconnect on context deadline to avoid data ordering
		// problems with subsequent read/writes. Re-establish the
		// connection later.
		cc.Close()
		<-rc

		return nil, ctx.Err()
	case v := <-rc:
		switch v.(type) {
		case error:
			return nil, v.(error)
		case *Message:
			return v.(*Message), nil
		default:
			// This is a programmer error.
			return nil, fmt.Errorf("%v: invalid packet read", errEncoding)
		}
	}
}

func (cc *clientConn) asyncPacketWrite(m *Message) <-chan error {
	r := make(chan error, 1)
	buf := bytes.NewBuffer([]byte{})

	go func() {
		defer close(r)
		b, err := m.encode()
		if err != nil {
			r <- err
			return
		}

		// Write the packet length
		if err := safePutUint32(buf, len(b)); err != nil {
			r <- err
			return
		}

		// Write the payload
		_, err = buf.Write(b)
		if err != nil {
			r <- err
			return
		}
		_, err = cc.conn.Write(buf.Bytes())
		r <- err
	}()

	return r
}

func (cc *clientConn) asyncPacketRead() <-chan any {
	r := make(chan any, 1)

	go func() {
		defer close(r)
		m := NewMessage()

		buf := make([]byte, headerLength)
		_, err := io.ReadFull(cc.conn, buf)
		if err != nil {
			r <- err
			return
		}
		pl := binary.BigEndian.Uint32(buf)

		buf = make([]byte, int(pl))
		_, err = io.ReadFull(cc.conn, buf)
		if err != nil {
			r <- err
			return
		}

		if err := m.decode(buf); err != nil {
			r <- err
			return
		}

		r <- m
	}()

	return r
}
