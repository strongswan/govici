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
	"iter"
	"net"
	"sync"
)

// Session is a vici client session.
type Session struct {
	// Only one command can be active on the transport at a time,
	// but events may get raised at any time while registered, even
	// during an active command request command. So, give session two
	// transports: one is locked with mutex during use, e.g. command
	// requests (including streamed requests), and the other is used
	// for listening to registered events.
	mu sync.Mutex
	cc *clientConn1

	el *eventListener

	// Session options.
	*sessionOpts
}

// NewSession returns a new vici session.
func NewSession(opts ...SessionOption) (*Session, error) {
	s := &Session{
		// Set default session opts before applying
		// the opts passed by the caller.
		sessionOpts: &sessionOpts{
			network: sessionDefaultNetwork,
			addr:    sessionDefaultAddr,
			dialer:  (&net.Dialer{}).DialContext,
			conn:    nil,
		},
	}

	for _, opt := range opts {
		opt.apply(s.sessionOpts)
	}

	cc, err := s.newClientConn()
	if err != nil {
		return nil, err
	}
	s.cc = cc

	elt, err := s.newClientConn()
	if err != nil {
		return nil, err
	}

	s.el = newEventListener(elt)

	return s, nil
}

// newClientConn creates a clientConn based on the session options.
func (s *Session) newClientConn() (*clientConn1, error) {
	// Check if a net.Conn was supplied already (testing only).
	if s.conn != nil {
		return &clientConn1{conn: s.conn}, nil
	}

	cc := &clientConn1{
		network: s.network,
		addr:    s.addr,
		dialer:  s.dialer,
		conn:    nil,
	}

	if err := cc.dial(context.Background()); err != nil {
		return nil, err
	}

	return cc, nil
}

// Close closes the vici session.
func (s *Session) Close() error {
	if err := s.el.Close(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cc != nil {
		if err := s.cc.Close(); err != nil {
			return err
		}

		s.cc = nil
	}

	return nil
}

// SessionOption is used to specify additional options
// to a Session.
type SessionOption interface {
	apply(*sessionOpts)
}

type sessionOpts struct {
	// Network and address to use to connect to the vici socket,
	// defaults to "unix" & "/var/run/charon.vici".
	network string
	addr    string

	// The context dial func to use when dialing the charon socket.
	dialer func(ctx context.Context, network, addr string) (net.Conn, error)

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
		so.network = "unix"
		so.addr = path
	})
}

// WithAddr specifies the network and address of the socket that charon
// is listening on. If this option is not specified, the default
// path, /var/run/charon.vici, is used.
//
// As the protocol itself currently does not provide any security or
// authentication properties, it is recommended to run it over a UNIX
// socket with appropriate permissions.
func WithAddr(network, addr string) SessionOption {
	return newFuncSessionOption(func(so *sessionOpts) {
		so.network = network
		so.addr = addr
	})
}

// WithDialContext specifies the dial func to use when dialing the charon socket.
func WithDialContext(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) SessionOption {
	return newFuncSessionOption(func(so *sessionOpts) {
		so.dialer = dialer
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
//
// Deprecated: Use the Call method instead. CommandRequest will be removed in a future version.
func (s *Session) CommandRequest(cmd string, msg *Message) (*Message, error) {
	return s.Call(context.Background(), cmd, msg)
}

// Call makes a command request to the server, and returns the server's response.
// The command is specified by cmd, and its arguments are provided by in. If there is an
// error communicating with the daemon, a nil Message and non-nil error are returned. If
// the command fails, the response Message is returned along with the error returned by
// Message.Err.
//
// The provided context must be non-nil, and can be used to interrupt blocking network I/O
// involved in the command request.
func (s *Session) Call(ctx context.Context, cmd string, in *Message) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx == nil {
		return nil, errors.New("ctx cannot be nil")
	}

	if s.cc == nil {
		return nil, errors.New("session closed")
	}

	if in == nil {
		in = NewMessage()
	}
	in.header = &header{
		ptype: pktCmdRequest,
		name:  cmd,
	}

	if err := s.cc.packetWrite(ctx, in); err != nil {
		return nil, err
	}

	p, err := s.cc.packetRead(ctx)
	if err != nil {
		return nil, err
	}

	if p.header.ptype != pktCmdResponse {
		return nil, fmt.Errorf("%v: %v", errUnexpectedResponse, p.header.ptype)
	}

	return p, p.Err()
}

// StreamedCommandRequest sends a streamed command request to the server. StreamedCommandRequest
// behaves like CommandRequest, but accepts an event argument, which specifies the event type
// to stream while the command request is active. The complete stream of messages received from
// the server is returned once the request is complete.
//
// Deprecated: Use the CallStreaming method instead. StreamedCommandRequest will be removed in
// a future version.
func (s *Session) StreamedCommandRequest(cmd string, event string, msg *Message) ([]*Message, error) {
	resp, err := s.CallStreaming(context.Background(), cmd, event, msg)
	if err != nil {
		return nil, err
	}

	messages := make([]*Message, 0)

	for m := range resp {
		messages = append(messages, m)
	}

	return messages, nil
}

// CallStreaming makes a command request to the server which, involves streaming a given event type
// until the command is complete. The command and event names are specified with cmd and event, and
// the command arguments are provided by in. If there is an error during the initial command request
// or event registration, no response messages are returned and a non-nil error is returned.
//
// When the initial command request and event registration are successful, an iterator is returned
// which will yield response messages, and possibly errors, as they are received from the server.
//
// The provided context must be non-nil, and can be used to interrupt blocking network I/O
// involved in the command request.
func (s *Session) CallStreaming(ctx context.Context, cmd string, event string, in *Message) (seq iter.Seq2[*Message, error], err error) {
	s.mu.Lock()
	defer func() {
		if err != nil {
			s.mu.Unlock()
		}
	}()

	if ctx == nil {
		return nil, errors.New("ctx cannot be nil")
	}

	if s.cc == nil {
		return nil, errors.New("session closed")
	}

	if in == nil {
		in = NewMessage()
	}
	in.header = &header{
		ptype: pktCmdRequest,
		name:  cmd,
	}

	if err := s.eventRegister(ctx, event); err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			// nolint
			s.eventUnregister(ctx, event)
		}
	}()

	if err := s.cc.packetWrite(ctx, in); err != nil {
		return nil, err
	}

	return func(yield func(*Message, error) bool) {
		defer s.mu.Unlock()
		// nolint
		defer s.eventUnregister(ctx, event)

		for {
			p, err := s.cc.packetRead(ctx)
			if err != nil {
				yield(nil, err)
				return
			}

			switch p.header.ptype {
			case pktEvent:
				if !yield(p, p.Err()) {
					return
				}
			case pktCmdResponse:
				yield(p, p.Err())
				return // End of event stream
			default:
				yield(nil, fmt.Errorf("%v: %v", errUnexpectedResponse, p.header.ptype))
				return
			}
		}
	}, nil
}

// Subscribe registers the session to listen for all events given. To receive
// events that are registered here, use NotifyEvents. An error is returned if
// Subscribe is not able to register the given events with the charon daemon.
func (s *Session) Subscribe(events ...string) error {
	return s.el.registerEvents(events)
}

// Unsubscribe unregisters the given events, so the session will no longer
// receive events of the given type. If a given event is not valid, an error
// is retured.
func (s *Session) Unsubscribe(events ...string) error {
	return s.el.unregisterEvents(events, false)
}

// UnsubscribeAll unregisters all events that the session is currently
// subscribed to.
func (s *Session) UnsubscribeAll() error {
	return s.el.unregisterEvents(nil, true)
}

// NotifyEvents registers c for writing received events. The Session must first
// subscribe to events using the Subscribe method.
//
// Writes to c will not block: the caller must ensure that c has sufficient
// buffer space to keep up with the expected event rate. If the write to c
// would block, the event is discarded.
//
// NotifyEvents may be called multiple times with different channels: each
// channel will indepedently receive a copy of each event received by the
// Session.
//
// When the Session is Close()'d, or the event listener otherwise exits, e.g.
// due to the daemon stopping or restarting, c will be closed to indicate
// that no more events will be passed to it.
func (s *Session) NotifyEvents(c chan<- Event) {
	s.el.notify(c)
}

// StopEvents stops writing received events to c.
func (s *Session) StopEvents(c chan<- Event) {
	s.el.stop(c)
}

func (s *Session) eventRequest(ctx context.Context, ptype uint8, event string) error {
	p := &Message{
		header: &header{
			ptype: ptype,
			name:  event,
		},
	}

	if err := s.cc.packetWrite(ctx, p); err != nil {
		return err
	}

	p, err := s.cc.packetRead(ctx)
	if err != nil {
		return err
	}

	switch p.header.ptype {
	case pktEventConfirm:
		return nil
	case pktEventUnknown:
		return fmt.Errorf("%v: %v", errEventUnknown, event)
	default:
		return fmt.Errorf("%v: %v", errUnexpectedResponse, p.header.ptype)
	}
}

func (s *Session) eventRegister(ctx context.Context, event string) error {
	return s.eventRequest(ctx, pktEventRegister, event)
}

func (s *Session) eventUnregister(ctx context.Context, event string) error {
	return s.eventRequest(ctx, pktEventUnregister, event)
}
