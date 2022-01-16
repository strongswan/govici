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
	"io"
	"sync"
	"time"
)

var (
	// Event listener channel was closed
	errChannelClosed = errors.New("vici: event listener channel closed")
)

type eventListener struct {
	*transport

	// Event channel and the events it's listening for.
	ec chan Event

	// Lock events when registering and unregistering.
	mu     sync.Mutex
	events []string

	// Packet channel used to communicate event registration
	// results.
	pc chan *packet

	muChans sync.Mutex
	chans   map[chan<- Event]struct{}
}

// Event represents an event received by a Session sent from the
// charon daemon. It contains an associated Message and corresponds
// to one of the event types registered with Session.Listen.
type Event struct {
	// Name is the event type name as specified by the
	// charon server, such as "ike-updown" or "log".
	Name string

	// Message is the Message associated with this event.
	Message *Message

	// Timestamp holds the timestamp of when the client
	// received the event.
	Timestamp time.Time
}

func newEventListener(t *transport) *eventListener {
	el := &eventListener{
		transport: t,
		ec:        make(chan Event, 16),
		pc:        make(chan *packet, 4),
		chans:     make(map[chan<- Event]struct{}),
	}

	go el.listen()

	return el
}

// Close closes the event channel.
func (el *eventListener) Close() error {
	// This call interacts with charon, so get it
	// done first. Then, we can stop the listen
	// goroutine.
	if err := el.unregisterEvents(nil, true); err != nil {
		return err
	}

	el.conn.Close()

	return nil
}

// listen is responsible for receiving all packets from the daemon. This includes
// not only event packets, but event registration confirmations/errors.
func (el *eventListener) listen() {
	// Clean up the event channel when this loop is closed. This
	// ensures any active NextEvent callers return.
	defer close(el.ec)
	defer close(el.pc)

	for {
		p, err := el.recv()
		if err != nil {
			return
		}

		ts := time.Now()

		switch p.ptype {
		case pktEvent:
			e := Event{
				Name:      p.name,
				Message:   p.msg,
				Timestamp: ts,
			}

			el.ec <- e
			el.dispatch(e)

		// These SHOULD be in response to event registration
		// requests from the event listener. Forward them over
		// the packet channel.
		case pktEventConfirm, pktEventUnknown:
			el.pc <- p
		}
	}
}

func (el *eventListener) notify(c chan<- Event) {
	el.muChans.Lock()
	defer el.muChans.Unlock()

	el.chans[c] = struct{}{}
}

func (el *eventListener) stop(c chan<- Event) {
	el.muChans.Lock()
	defer el.muChans.Unlock()

	delete(el.chans, c)
}

func (el *eventListener) dispatch(e Event) {
	el.muChans.Lock()
	defer el.muChans.Unlock()

	for c := range el.chans {
		select {
		case c <- e:
		default:
		}
	}
}

func (el *eventListener) nextEvent(ctx context.Context) (Event, error) {
	select {
	case <-ctx.Done():
		return Event{}, ctx.Err()

	case e, ok := <-el.ec:
		if !ok {
			return Event{}, errChannelClosed
		}

		return e, nil
	}
}

func (el *eventListener) registerEvents(events []string) error {
	el.mu.Lock()
	defer el.mu.Unlock()

	for _, event := range events {
		// Check if the event is already registered.
		exists := false

		for _, registered := range el.events {
			if event == registered {
				exists = true
				// Break out of the inner loop, this
				// event is already registered.
				break
			}
		}

		// Check the next event given...
		if exists {
			continue
		}

		if err := el.eventRegisterUnregister(event, true); err != nil {
			return err
		}

		// Add the event to the list of registered events.
		el.events = append(el.events, event)
	}

	return nil
}

func (el *eventListener) unregisterEvents(events []string, all bool) error {
	el.mu.Lock()
	defer el.mu.Unlock()

	if all {
		events = el.events
	}

	for _, e := range events {
		if err := el.eventRegisterUnregister(e, false); err != nil {
			return err
		}

		for i, registered := range el.events {
			if e != registered {
				continue
			}

			el.events = append(el.events[:i], el.events[i+1:]...)

			// Break from the inner loop, we found the event
			// in the list of registered events.
			break
		}
	}

	return nil
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

	p, ok := <-el.pc
	if !ok {
		return nil, io.ErrClosedPipe
	}

	return p, nil
}
