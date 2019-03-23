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
	"fmt"
)

func (s *Session) sendRequest(cmd string, msg *Message) (*Message, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	p, err := s.cmdTransportCommunicate(newPacket(pktCmdRequest, cmd, msg))
	if err != nil {
		return nil, err
	}

	if p.ptype != pktCmdResponse {
		return nil, fmt.Errorf("unexpected response type: %v", p.ptype)
	}

	if p.msg.CheckSuccess() != nil {
		return nil, err
	}

	return p.msg, nil
}

func (s *Session) sendStreamedRequest(cmd string, event string, msg *Message) (*MessageStream, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	err := s.streamEventRegisterUnregister(event, true)
	if err != nil {
		return nil, err
	}

	return s.handleStreamedRequest(cmd, event, msg)
}

func (s *Session) handleStreamedRequest(cmd, event string, msg *Message) (*MessageStream, error) {
	// nolint
	defer s.streamEventRegisterUnregister(event, false)

	messages := make([]*Message, 0)

	p := newPacket(pktCmdRequest, cmd, msg)

	err := s.ctr.send(p)
	if err != nil {
		return nil, err
	}

	for {
		p, err = s.ctr.recv()
		if err != nil {
			return nil, err
		}

		if p.ptype != pktEvent {
			break
		}

		messages = append(messages, p.msg)
	}

	// Packet type was not event, check if it was command response
	if p.ptype != pktCmdResponse {
		err := fmt.Errorf("unexpected packet type in message stream: %v", p.ptype)
		return nil, err
	}
	messages = append(messages, p.msg)

	return &MessageStream{messages}, nil
}

// streamEventRegisterUnregister will (un)register the given event type, based on the register boolean.
// This should only be used internally from within functions that have the session lock.
func (s *Session) streamEventRegisterUnregister(event string, register bool) error {
	ptype := pktEventRegister
	if !register {
		ptype = pktEventUnregister
	}

	p, err := s.cmdTransportCommunicate(newPacket(ptype, event, nil))
	if err != nil {
		return err
	}

	if p.ptype == pktEventUnknown {
		return fmt.Errorf("unknown event type '%v'", event)
	}

	if p.ptype != pktEventConfirm {
		return fmt.Errorf("unexpected packet type: expected %v but received %v", pktEventConfirm, p.ptype)
	}

	return nil
}

// cmdTransportCommunicate is used to send command requests over the dedicated
// dedicated transport. It should only be used within a function with the session
// lock.
func (s *Session) cmdTransportCommunicate(pkt *packet) (*packet, error) {
	err := s.ctr.send(pkt)
	if err != nil {
		return nil, err
	}

	p, err := s.ctr.recv()
	if err != nil {
		return nil, err
	}

	return p, err
}
