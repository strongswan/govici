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
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

const (
	// Default unix socket path
	defaultSocketPath = "/var/run/charon.vici"

	// Each segment is prefixed by a 4-byte header in network oreder
	headerLength = 4
)

var (
	// Generic transport error
	errTransport = errors.New("vici: transport error")
)

type transport struct {
	conn net.Conn
}

func (t *transport) send(pkt *packet) error {
	buf := bytes.NewBuffer([]byte{})

	b, err := pkt.bytes()
	if err != nil {
		return err
	}

	// Write the packet length
	pl := make([]byte, headerLength)
	binary.BigEndian.PutUint32(pl, uint32(len(b)))
	_, err = buf.Write(pl)
	if err != nil {
		return fmt.Errorf("%v: %v", errTransport, err)
	}

	// Write the payload
	_, err = buf.Write(b)
	if err != nil {
		return fmt.Errorf("%v: %v", errTransport, err)
	}

	_, err = t.conn.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("%v: %v", errTransport, err)
	}

	return nil
}

func (t *transport) recv() (*packet, error) {
	buf := make([]byte, headerLength)

	_, err := io.ReadFull(t.conn, buf)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil, ne
		}

		return nil, fmt.Errorf("%v: %v", errTransport, err)
	}
	pl := binary.BigEndian.Uint32(buf)

	buf = make([]byte, int(pl))
	_, err = io.ReadFull(t.conn, buf)
	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			return nil, ne
		}

		return nil, fmt.Errorf("%v: %v", errTransport, err)
	}

	p := &packet{}
	err = p.parse(buf)
	if err != nil {
		return nil, err
	}

	return p, nil
}
