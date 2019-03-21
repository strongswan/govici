// Package vici implements a strongSwan vici protocol client
package vici

import (
	"bytes"
	"errors"
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

// A packet has a required type (an 8-bit identifier), a name (only required for named types),
// and and an optional message field.
type packet struct {
	ptype uint8
	name  string

	msg *message
}

func newPacket(ptype uint8, name string, msg *message) *packet {
	return &packet{
		ptype: ptype,
		name:  name,
		msg:   msg,
	}
}

// isNamed returns a bool indicating the packet is a named type
func (p *packet) isNamed() bool {
	n := map[uint8]bool{
		pktCmdRequest:      true,
		pktCmdResponse:     false,
		pktCmdUnkown:       false,
		pktEventRegister:   true,
		pktEventUnregister: true,
		pktEventConfirm:    false,
		pktEventUnknown:    false,
		pktEvent:           true,
	}

	return n[p.ptype]
}

// bytes formats the packet and returns it as a byte slice
func (p *packet) bytes() ([]byte, error) {
	// Create a new buffer with the first byte indicating the packet type
	buf := bytes.NewBuffer([]byte{p.ptype})

	// Write the name, preceeded by its length
	if p.isNamed() {
		err := buf.WriteByte(uint8(len(p.name)))
		if err != nil {
			return []byte{}, err
		}
	}

	_, err := buf.WriteString(p.name)
	if err != nil {
		return []byte{}, err
	}

	if p.msg != nil {
		b, err := p.msg.encode()
		if err != nil {
			return []byte{}, err
		}

		_, err = buf.Write(b)
		if err != nil {
			return []byte{}, err
		}
	}

	return buf.Bytes(), nil
}

// parse will parse the given bytes and populate its fields with that data
func (p *packet) parse(data []byte) error {
	buf := bytes.NewBuffer(data)

	// Read the packet type
	b, err := buf.ReadByte()
	if err != nil {
		return err
	}
	p.ptype = b

	if p.isNamed() {
		// Get the length of the name
		l, err := buf.ReadByte()
		if err != nil {
			return nil
		}

		// Read the name
		name := buf.Next(int(l))
		if len(name) != int(l) {
			return errors.New("expected name length does not match actual length")
		}
		p.name = string(name)
	}

	// Decode the message field
	m := newMessage()
	err = m.decode(buf.Bytes())
	if err != nil {
		return err
	}
	p.msg = m

	return nil
}
