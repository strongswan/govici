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
	"net"
	"time"
)

var (
	// Event listener channel was closed
	errChannelClosed = errors.New("vici: event listener channel closed")
)

type eventListener struct {
	*transport

	// Internal Context and CancelFunc used to stop the
	// listen loop.
	ctx    context.Context
	cancel context.CancelFunc

	// Used in destroy() to signal that cleanup has finished.
	//
	// This pair is separate than the above ctx and cancel since
	// cancel is used as the entry point to destroy, and so ctx.Done()
	// would return before destroy has finished.
	dctx    context.Context
	dcancel context.CancelFunc

	// Event channel and the events it's listening for.
	ec     chan Event
	events []string
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

	err error
}

func newEventListener(t *transport) *eventListener {
	ctx, cancel := context.WithCancel(context.Background())
	dctx, dcancel := context.WithCancel(context.Background())

	return &eventListener{
		transport: t,
		ctx:       ctx,
		cancel:    cancel,
		dctx:      dctx,
		dcancel:   dcancel,
		ec:        make(chan Event, 16),
	}
}

// Close closes the event channel.
func (el *eventListener) Close() error {
	// Cancel the event listener context, and
	// wait for the destroy context to be done.
	el.cancel()

	<-el.dctx.Done()

	return nil
}

func (el *eventListener) isActive() bool {
	if el.dctx == nil {
		return false
	}

	select {
	case <-el.dctx.Done():
		// This case means the event listener has been
		// destroy()'d, so it's no longer active.
		return false

	default:
		// If these contexts are still active, then so is the
		// event listener.
		return true
	}
}

func (el *eventListener) destroy() {
	// This call interacts with charon, so get it
	// done first. Then, the transport conn can
	// be closed.
	el.unregisterEvents(el.events)
	el.conn.Close()

	// Close event channel. This ensures that any active
	// calls to NextEvent will return.
	close(el.ec)

	// Finally, signal that destroy is finished. This MUST
	// be done last, as it acts as a signal to Close that
	// cleanup is done.
	el.dcancel()
}

func (el *eventListener) listen(events []string) (err error) {
	err = el.registerEvents(events)
	if err != nil {
		return err
	}
	el.events = events

	go func() {
		defer el.destroy()

		for {
			select {
			case <-el.ctx.Done():
				// Closer context was cancelled.
				return

			default:
				var e Event

				// Set a read deadline so that this loop can continue
				// at a reasonable pace. If the error is a timeout,
				// do not send it on the event channel.
				_ = el.conn.SetReadDeadline(time.Now().Add(time.Second))

				p, err := el.recv()
				if err != nil {
					if ne, ok := err.(net.Error); ok && ne.Timeout() {
						continue
					}
					e.err = err

					// Send error event and continue in loop.
					el.ec <- e
					continue
				}

				if p.ptype == pktEvent {
					e.Name = p.name
					e.Message = p.msg
					el.ec <- e
				}
			}
		}
	}()

	return nil
}

func (el *eventListener) nextEvent() (Event, error) {
	e := <-el.ec
	if e.Message == nil && e.err == nil {
		return Event{}, errChannelClosed
	}

	return e, e.err
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

	// Refresh deadline for communication.
	err = el.conn.SetReadDeadline(time.Now().Add(time.Second))
	if err != nil {
		return nil, err
	}

	p, err := el.recv()
	if err != nil {
		return nil, err
	}

	return p, nil
}
