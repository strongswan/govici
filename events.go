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
	"errors"
	"fmt"
)

const (
	// Log event is issued to registered clients for each debug
	// log message. This event is not associated with a command.
	Log = "log"

	// ControlLog event is issued for log events during active initiate
	// or terminate commands. It is issued only to clients currently having such a command active.
	ControlLog = "control-log"

	// ListSA event is issued to stream IKE_SAs during an active list-sas command.
	ListSA = "list-sa"

	// ListPolicy event is issued to stream installed policies during an active list-policies command.
	ListPolicy = "list-policy"

	// ListConn event is issued to stream loaded connection during an active list-conns command.
	ListConn = "list-conn"

	// ListCert event is issued to stream loaded certificates during an active list-certs command.
	ListCert = "list-cert"

	// ListAuthority event is issued to stream loaded certification authority information
	// during an active_list-authorities_ command.
	ListAuthority = "list-authority"

	// IKEUpdown event is issued when an IKE_SA is established or terminated.
	IKEUpdown = "ike-updown"

	// IKERekey event is issued when an IKE_SA is rekeyed.
	IKERekey = "ike-rekey"

	// ChildUpdown event is issued when a CHILD_SA is established or terminated.
	ChildUpdown = "child-updown"

	// ChildRekey event is issued when a CHILD_SA is rekeyed.
	ChildRekey = "child-rekey"
)

var (
	// Event listener channel was closed
	errChannelClosed = errors.New("vici: event listener channel closed")
)

type eventError struct{ error }

type eventListener struct {
	*transport

	mc chan *Message
}

func newEventListener(t *transport) *eventListener {
	return &eventListener{
		transport: t,
		mc:        make(chan *Message),
	}
}

func (el *eventListener) nextEvent() (*Message, error) {
	m := <-el.mc
	if m == nil {
		return nil, errChannelClosed
	}

	return m, nil
}

func (el *eventListener) safeListen(events []string) (err error) {
	err = el.registerEvents(events)
	if err != nil {
		return err
	}
	defer el.unregisterEvents(events)
	defer func() {
		if r := recover(); r != nil {
			if ee, ok := r.(eventError); ok {
				err = ee.error
			} else {
				panic(r)
			}
		}
	}()

	el.listen()

	return nil
}

func (el *eventListener) listen() {
	defer close(el.mc)

	for {
		p, err := el.recv()
		if err != nil {
			panic(eventError{err})
		}

		if p.ptype == pktEvent {
			el.mc <- p.msg
		}
	}
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

	p, err := el.recv()
	if err != nil {
		return nil, err
	}

	return p, nil
}
