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
	"fmt"
	"io"
	"sync"
	"time"
)

type eventListener struct {
	cc *clientConn

	// Lock events when registering and unregistering.
	mu     sync.Mutex
	events []string

	// Packet channel used to communicate event registration
	// results.
	pc chan *Message

	muChans sync.Mutex
	chans   map[chan<- Event]struct{}

	// Lazily start the listen loop on the first call to registerEvents.
	listenOnce sync.Once
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

func newEventListener(cc *clientConn) *eventListener {
	el := &eventListener{
		cc:    cc,
		pc:    make(chan *Message, 4),
		chans: make(map[chan<- Event]struct{}),
	}

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

	el.cc.conn.Close()

	return nil
}

// listen is responsible for receiving all packets from the daemon. This includes
// not only event packets, but event registration confirmations/errors.
func (el *eventListener) listen() {
	defer close(el.pc)
	defer el.closeAllChans()

	for {
		m, err := el.cc.packetRead(context.Background())
		if err != nil {
			return
		}

		ts := time.Now()

		switch m.header.ptype {
		case pktEvent:
			e := Event{
				Name:      m.header.name,
				Message:   m,
				Timestamp: ts,
			}

			el.dispatch(e)

		// These SHOULD be in response to event registration
		// requests from the event listener. Forward them over
		// the packet channel.
		case pktEventConfirm, pktEventUnknown:
			el.pc <- m
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

func (el *eventListener) closeAllChans() {
	el.muChans.Lock()
	defer el.muChans.Unlock()

	for c := range el.chans {
		close(c)
	}
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

func (el *eventListener) registerEvents(events []string) error {
	el.listenOnce.Do(func() { go el.listen() })

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

		if err := el.register(event); err != nil {
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

	for _, event := range events {
		if err := el.unregister(event); err != nil {
			return err
		}

		for i, registered := range el.events {
			if event != registered {
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

func (el *eventListener) eventRequest(ptype uint8, event string) error {
	m := &Message{
		header: &struct {
			ptype uint8
			name  string
		}{
			ptype: ptype,
			name:  event,
		},
	}

	if err := el.cc.packetWrite(context.Background(), m); err != nil {
		return err
	}

	// The response packet is read by listen(), and written over pc.
	m, ok := <-el.pc
	if !ok {
		return io.ErrClosedPipe
	}

	switch m.header.ptype {
	case pktEventConfirm:
		return nil
	case pktEventUnknown:
		return fmt.Errorf("%v: %v", errEventUnknown, event)
	default:
		return fmt.Errorf("%v: %v", errUnexpectedResponse, m.header.ptype)
	}
}

func (el *eventListener) register(event string) error {
	return el.eventRequest(pktEventRegister, event)
}

func (el *eventListener) unregister(event string) error {
	return el.eventRequest(pktEventUnregister, event)
}
