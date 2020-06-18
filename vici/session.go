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

// Package vici implements a strongSwan vici protocol client. The Go package is
// documented here. For a complete overview and specification of the vici
// protocol visit:
//
//     https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html
//
package vici

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
)

var (
	// Event listener errors
	errNoEventListener     = errors.New("vici: event listener is not active")
	errEventListenerExists = errors.New("vici: event listener already exists")
)

// Session is a vici client session.
type Session struct {
	// Only one command can be active on the transport at a time,
	// but events may get raised at any time while registered, even
	// during an active command request command. So, give session two
	// transports: one is locked with mutex during use, e.g. command
	// requests (including streamed requests), and the other is used
	// for listening to registered events.
	mux sync.Mutex
	ctr *transport

	// Allow many readers, i.e. NextEvent callers, to try to read from
	// event listener. Writer lock is for creation and destruction of
	// the event listener.
	emux sync.RWMutex
	el   *eventListener

	// Session options.
	*sessionOpts
}

// NewSession returns a new vici session.
func NewSession(opts ...SessionOption) (*Session, error) {
	s := &Session{
		sessionOpts: &sessionOpts{},
	}

	for _, opt := range opts {
		opt.apply(s.sessionOpts)
	}

	ctr, err := s.newTransport()
	if err != nil {
		return nil, err
	}

	s.ctr = ctr

	return s, nil
}

// newTransport creates a transport based on the session options.
func (s *Session) newTransport() (*transport, error) {
	// Check if a net.Conn was supplied already (testing only).
	if s.conn != nil {
		return &transport{conn: s.conn}, nil
	}

	if s.path == "" {
		s.path = defaultSocketPath
	}

	conn, err := net.Dial("unix", s.path)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errTransport, err)
	}

	t := &transport{
		conn: conn,
	}

	return t, nil
}

// Close closes the vici session.
func (s *Session) Close() error {
	// We can use the reader's lock to see if
	// the event listener exists, and to close it.
	//
	// Use the writer's lock later to set the event
	// listener to nil.
	s.emux.RLock()
	if s.el != nil {
		if err := s.el.Close(); err != nil {
			return err
		}
	}
	s.emux.RUnlock()

	s.emux.Lock()
	s.el = nil
	s.emux.Unlock()

	s.mux.Lock()
	if s.ctr != nil {
		if err := s.ctr.conn.Close(); err != nil {
			return err
		}

		s.ctr = nil
	}
	s.mux.Unlock()

	return nil
}

// SessionOption is used to specify additional options
// to a Session.
type SessionOption interface {
	apply(*sessionOpts)
}

type sessionOpts struct {
	// Path to vici socket, defaults to /var/run/charon.vici.
	path string

	// A net.Conn to use, instead of dialing a unix socket.
	//
	// This is only used for testing purposes.
	conn net.Conn
}

type funcSessionOption struct {
	f func(*sessionOpts)
}

func (fso *funcSessionOption) apply(s *sessionOpts) {
	fso.f(s)
}

func newFuncSessionOption(f func(*sessionOpts)) *funcSessionOption {
	return &funcSessionOption{f}
}

// WithSocketPath specifies the path of the socket that charon
// is listening on. If this option is not specified, the default
// path, /var/run/charon.vici, is used.
func WithSocketPath(path string) SessionOption {
	return newFuncSessionOption(func(so *sessionOpts) {
		so.path = path
	})
}

// withTestConn is a SessionOption used in testing to supply a net.Conn
// without actually dialing a unix socket.
func withTestConn(conn net.Conn) SessionOption {
	return newFuncSessionOption(func(so *sessionOpts) {
		so.conn = conn
	})
}

// CommandRequest sends a command request to the server, and returns the server's response.
// The command is specified by cmd, and its arguments are provided by msg. If there is an
// error communicating with the daemon, a nil Message and non-nil error are returned. If
// the command fails, the response Message is returned along with the error returned by
// Message.Err.
func (s *Session) CommandRequest(cmd string, msg *Message) (*Message, error) {
	resp, err := s.sendRequest(cmd, msg)
	if err != nil {
		return nil, err
	}

	return resp, resp.Err()
}

// StreamedCommandRequest sends a streamed command request to the server. StreamedCommandRequest
// behaves like CommandRequest, but accepts an event argument, which specifies the event type
// to stream while the command request is active. The complete stream of messages received from
// the server is returned once the request is complete.
func (s *Session) StreamedCommandRequest(cmd string, event string, msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest(cmd, event, msg)
}

// Listen registers the session to listen for all events given. Listen returns when the
// event channel is closed, or the given context is cancelled. To receive events that
// are registered here, use NextEvent. An error is returned if Listen is called while
// Session already has an event listener registered.
func (s *Session) Listen(ctx context.Context, events ...string) error {
	s.emux.Lock()
	defer s.emux.Unlock()

	if err := s.maybeCreateEventListener(ctx); err != nil {
		return err
	}

	return s.el.listen(events)
}

func (s *Session) maybeCreateEventListener(ctx context.Context) error {
	if s.el != nil {
		if s.el.isActive() {
			return errEventListenerExists
		}
	}

	// Safe to create a new event listener...
	elt, err := s.newTransport()
	if err != nil {
		return err
	}

	s.el = newEventListener(ctx, elt)

	return nil
}

// NextEvent returns the next event received by the session event listener.  NextEvent is a
// blocking call. If there is no event in the event buffer, NextEvent will wait to return until
// a new event is received. An error is returned if the event channel is closed.
func (s *Session) NextEvent() (Event, error) {
	s.emux.RLock()
	defer s.emux.RUnlock()

	if s.el == nil {
		return Event{}, errNoEventListener
	}

	return s.el.nextEvent()
}
