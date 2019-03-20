// Package vici implements a strongSwan vici protocol client
package vici

import (
	"bytes"
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

// bytes formats the packet and returns it as a byte slice
func (p *packet) bytes() ([]byte, error) {
	// Create a new buffer with the first byte indicating the packet type
	buf := bytes.NewBuffer([]byte{p.ptype})

	// Write the name, preceeded by its length
	err := buf.WriteByte(uint8(len(p.name)))
	if err != nil {
		return []byte{}, err
	}

	_, err := buf.WriteString(p.name)
	if err != nil {
		return []byte{}, err
	}

	if msg != nil {
		b, err := msg.encode()
		if err != nil {
			return []byte{}, err
		}

		_, err := buf.Write(b)
		if err != nil {
			return []byte{}, err
		}
	}

	return buf.Bytes(), nil
}
