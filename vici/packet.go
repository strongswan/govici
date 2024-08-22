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
	"errors"
	"fmt"
)

const (
	// A name request message
	pktCmdRequest uint8 = iota

	// An unnamed response message for a request
	pktCmdResponse

	// An unnamed response if requested command is unknown
	pktCmdUnkown

	// A named event registration request
	pktEventRegister

	// A name event deregistration request
	pktEventUnregister

	// An unnamed response for successful event (de-)registration
	pktEventConfirm

	// An unnamed response if event (de-)registration failed
	pktEventUnknown

	// A named event message
	pktEvent
)

var (
	// Generic packet writing error
	errPacketWrite = errors.New("vici: error writing packet")

	// Generic packet parsing error
	errPacketParse = errors.New("vici: error parsing packet")

	errBadName = fmt.Errorf("%v: expected name length does not match actual length", errPacketParse)
)

// A packet has a required type (an 8-bit identifier), a name (only required for named types),
// and and an optional message field.
type packet struct {
	ptype uint8
	name  string

	msg *Message
	err error
}

func newPacket(ptype uint8, name string, msg *Message) *packet {
	return &packet{
		ptype: ptype,
		name:  name,
		msg:   msg,
	}
}

// isNamed returns a bool indicating the packet is a named type
func (p *packet) isNamed() bool {
	switch p.ptype {
	case /* Named packet types */
		pktCmdRequest,
		pktEventRegister,
		pktEventUnregister,
		pktEvent:

		return true

	case /* Un-named packet types */
		pktCmdResponse,
		pktCmdUnkown,
		pktEventConfirm,
		pktEventUnknown:

		return false
	}

	return false
}

// bytes formats the packet and returns it as a byte slice
func (p *packet) bytes() ([]byte, error) {
	// Create a new buffer with the first byte indicating the packet type
	buf := bytes.NewBuffer([]byte{p.ptype})

	// Write the name, preceded by its length
	if p.isNamed() {
		err := safePutUint8(buf, len(p.name))
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errPacketWrite, err)
		}

		_, err = buf.WriteString(p.name)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errPacketWrite, err)
		}
	}

	if p.msg != nil {
		b, err := p.msg.encode()
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(b)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errPacketWrite, err)
		}
	}

	return buf.Bytes(), nil
}

// parse will parse the given bytes and populate its fields with that data. If
// there is an error, then the error filed will be set.
func (p *packet) parse(data []byte) {
	buf := bytes.NewBuffer(data)

	// Read the packet type
	b, err := buf.ReadByte()
	if err != nil {
		p.err = fmt.Errorf("%v: %v", errPacketParse, err)
		return
	}
	p.ptype = b

	if p.isNamed() {
		// Get the length of the name
		l, err := buf.ReadByte()
		if err != nil {
			p.err = fmt.Errorf("%v: %v", errPacketParse, err)
			return
		}

		// Read the name
		name := buf.Next(int(l))
		if len(name) != int(l) {
			p.err = errBadName
			return
		}
		p.name = string(name)
	}

	// Decode the message field
	m := NewMessage()
	err = m.decode(buf.Bytes())
	if err != nil {
		p.err = err
		return
	}
	p.msg = m
}
