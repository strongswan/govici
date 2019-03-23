//
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
//

// Package vici implements a strongSwan vici protocol client
package vici

import (
	"bytes"
	"encoding/binary"
	"net"
)

const (
	// Default unix socket path
	viciSocket = "/var/run/charon.vici"

	// Each segment is prefixed by a 4-byte header in network oreder
	headerLength = 4

	// Maximum segment length is 512KB
	maxSegment = 512 * 1024
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
		return err
	}

	// Write the payload
	_, err = buf.Write(b)
	if err != nil {
		return err
	}

	_, err = t.conn.Write(buf.Bytes())

	return err
}

func (t *transport) recv() (*packet, error) {
	buf := make([]byte, headerLength)

	_, err := t.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	pl := binary.BigEndian.Uint32(buf)

	buf = make([]byte, int(pl))
	_, err = t.conn.Read(buf)
	if err != nil {
		return nil, err
	}

	p := &packet{}
	err = p.parse(buf)
	if err != nil {
		return nil, err
	}

	return p, nil
}
