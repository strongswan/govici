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
	"io"
	"net"
	"time"
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
	conn net.Conn
}

func (cc *clientConn) packetWrite(ctx context.Context, p *packet) error {
	if err := cc.conn.SetWriteDeadline(time.Time{}); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		err := cc.conn.SetWriteDeadline(time.Now())
		return errors.Join(err, ctx.Err())
	case err := <-cc.awaitPacketWrite(p):
		if err != nil {
			return err
		}
		return nil
	}
}

func (cc *clientConn) packetRead(ctx context.Context) (*packet, error) {
	if err := cc.conn.SetReadDeadline(time.Time{}); err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		err := cc.conn.SetReadDeadline(time.Now())
		return nil, errors.Join(err, ctx.Err())
	case p := <-cc.awaitPacketRead():
		if p.err != nil {
			return nil, p.err
		}
		return p, nil
	}
}

func (cc *clientConn) awaitPacketWrite(p *packet) <-chan error {
	r := make(chan error, 1)
	buf := bytes.NewBuffer([]byte{})

	go func() {
		defer close(r)
		b, err := p.bytes()
		if err != nil {
			r <- err
			return
		}

		// Write the packet length
		pl := make([]byte, headerLength)
		binary.BigEndian.PutUint32(pl, uint32(len(b)))
		_, err = buf.Write(pl)
		if err != nil {
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

func (cc *clientConn) awaitPacketRead() <-chan *packet {
	r := make(chan *packet, 1)

	go func() {
		defer close(r)
		p := &packet{}

		buf := make([]byte, headerLength)
		_, err := io.ReadFull(cc.conn, buf)
		if err != nil {
			p.err = err
			r <- p
			return
		}
		pl := binary.BigEndian.Uint32(buf)

		buf = make([]byte, int(pl))
		_, err = io.ReadFull(cc.conn, buf)
		if err != nil {
			p.err = err
			r <- p
			return
		}

		p.parse(buf)
		r <- p
	}()

	return r
}
