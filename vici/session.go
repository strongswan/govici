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
	"iter"
	"net"
	"time"
)

// Session is a vici client session.
type Session struct {
	cc *clientConn

	// Network and address to use to connect to the vici socket,
	// defaults to "unix" & "/var/run/charon.vici".
	network string
	addr    string

	// The context dial func to use when dialing the charon socket.
	dialer func(ctx context.Context, network, addr string) (net.Conn, error)
}

// NewSession returns a new vici session.
func NewSession(opts ...SessionOption) (*Session, error) {
	s := &Session{
		// Set default session opts before applying
		// the opts passed by the caller.
		network: sessionDefaultNetwork,
		addr:    sessionDefaultAddr,
		dialer:  (&net.Dialer{}).DialContext,
		cc:      nil,
	}

	for _, opt := range opts {
		opt.apply(s)
	}

	if s.cc != nil {
		// Testing only. A net.Conn was given.
		return s, nil
	}

	conn, err := s.dialer(context.Background(), s.network, s.addr)
	if err != nil {
		return nil, err
	}

	s.cc = newClientConn(conn)
	go s.cc.listen()

	return s, nil
}

// Close closes the vici session.
func (s *Session) Close() error {
	return s.cc.Close()
}

// SessionOption is used to specify additional options
// to a Session.
type SessionOption interface {
	apply(*Session)
}

type funcSessionOption struct {
	f func(*Session)
}

func (fso *funcSessionOption) apply(s *Session) {
	fso.f(s)
}

func newFuncSessionOption(f func(*Session)) *funcSessionOption {
	return &funcSessionOption{f}
}

// WithSocketPath specifies the path of the socket that charon
// is listening on. If this option is not specified, the default
// path, /var/run/charon.vici, is used.
func WithSocketPath(path string) SessionOption {
	return newFuncSessionOption(func(so *Session) {
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
	return newFuncSessionOption(func(so *Session) {
		so.network = network
		so.addr = addr
	})
}

// WithDialContext specifies the dial func to use when dialing the charon socket.
func WithDialContext(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) SessionOption {
	return newFuncSessionOption(func(so *Session) {
		so.dialer = dialer
	})
}

// withTestConn is a SessionOption used in testing to supply a net.Conn
// without actually dialing a unix socket.
func withTestConn(conn net.Conn) SessionOption {
	return newFuncSessionOption(func(so *Session) {
		so.cc = newClientConn(conn)
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
	s.cc.Lock()
	defer s.cc.Unlock()

	return s.cc.call(ctx, cmd, in)
}

// StreamedCommandRequest sends a streamed command request to the server. StreamedCommandRequest
// behaves like CommandRequest, but accepts an event argument, which specifies the event type
// to stream while the command request is active. The complete stream of messages received from
// the server is returned once the request is complete.
//
// Deprecated: Use the CallStreaming method instead. StreamedCommandRequest will be removed in
// a future version.
func (s *Session) StreamedCommandRequest(cmd string, event string, msg *Message) ([]*Message, error) {
	messages := make([]*Message, 0)

	for m, err := range s.CallStreaming(context.Background(), cmd, event, msg) {
		if err != nil {
			return nil, err
		}

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
func (s *Session) CallStreaming(ctx context.Context, cmd string, event string, in *Message) iter.Seq2[*Message, error] {
	s.cc.Lock()
	defer s.cc.Unlock()

	return s.cc.stream(ctx, cmd, event, in)
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

// Subscribe registers the session to listen for all events given. To receive
// events that are registered here, use NotifyEvents. An error is returned if
// Subscribe is not able to register the given events with the charon daemon.
func (s *Session) Subscribe(events ...string) error {
	s.cc.Lock()
	defer s.cc.Unlock()

	return s.cc.subscribe(context.Background(), events...)
}

// Unsubscribe unregisters the given events, so the session will no longer
// receive events of the given type. If a given event is not valid, an error
// is retured.
func (s *Session) Unsubscribe(events ...string) error {
	s.cc.Lock()
	defer s.cc.Unlock()

	return s.cc.unsubscribe(context.Background(), events...)
}

// UnsubscribeAll unregisters all events that the session is currently
// subscribed to.
func (s *Session) UnsubscribeAll() error {
	s.cc.Lock()
	defer s.cc.Unlock()

	return s.cc.unsubscribe(context.Background())
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
	s.cc.notify(c)
}

// StopEvents stops writing received events to c.
func (s *Session) StopEvents(c chan<- Event) {
	s.cc.unnotify(c)
}
