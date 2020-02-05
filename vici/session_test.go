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
	"log"
	"net"
	"testing"
)

func mockCharon(ctx context.Context) net.Conn {
	client, srvr := net.Pipe()

	go func() {
		defer func() {
			srvr.Close()
		}()

		tr := &transport{conn: srvr}

		for {
			select {
			case <-ctx.Done():
				return
			default:
				break
			}

			p, err := tr.recv()
			if err != nil {
				return
			}

			switch p.ptype {
			case pktEventRegister, pktEventUnregister:
				var ack *packet

				if p.name != "test-event" {
					ack = newPacket(pktEventUnknown, "", nil)
				} else {
					ack = newPacket(pktEventConfirm, "", nil)
				}

				err := tr.send(ack)
				if err != nil {
					return
				}

				if p.ptype == pktEventRegister {
					// Write one event message
					msg := NewMessage()
					err := msg.Set("test", "hello world!")
					if err != nil {
						log.Printf("Failed to set message field: %v", err)
					}
					event := newPacket(pktEvent, "test-event", msg)
					err = tr.send(event)
					if err != nil {
						log.Printf("Failed to send test-event message: %v", err)
					}
				}

			default:
				continue
			}
		}
	}()

	return client
}

func TestListenAndCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dctx, dcancel := context.WithCancel(context.Background())
	defer dcancel()

	conn := mockCharon(dctx)

	s, err := NewSession(withTestConn(conn))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer s.Close()

	if err := s.Listen(ctx, "test-event"); err != nil {
		t.Fatalf("Failed to start event listener: %v", err)
	}

	m, err := s.NextEvent()
	if err != nil {
		t.Fatalf("Unexpected error on NextEvent: %v", err)
	}

	if m.Get("test") != "hello world!" {
		t.Fatalf("Unexpected message: %v", m)
	}

	cancel()

	m, err = s.NextEvent()
	if err == nil {
		t.Fatalf("Expected error after closing listener, got message: %v", m)
	}
}

func TestListenAndCloseSession(t *testing.T) {
	dctx, dcancel := context.WithCancel(context.Background())
	defer dcancel()

	conn := mockCharon(dctx)

	s, err := NewSession(withTestConn(conn))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	defer s.Close()

	err = s.Listen(context.Background(), "test-event")
	if err != nil {
		t.Fatalf("Failed to start event listener: %v", err)
	}

	m, err := s.NextEvent()
	if err != nil {
		t.Fatalf("Unexpected error on NextEvent: %v", err)
	}

	if m.Get("test") != "hello world!" {
		t.Fatalf("Unexpected message: %v", m)
	}

	// Close session
	s.Close()

	m, err = s.NextEvent()
	if err == nil {
		t.Fatalf("Expected error after closing listener, got message: %v", m)
	}
}

func TestSessionClose(t *testing.T) {
	// Create a session without connecting to charon
	conn, _ := net.Pipe()

	s, err := NewSession(withTestConn(conn))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Unpexected error when closing Session: %v", err)
	}
}

func TestIdempotentSessionClose(t *testing.T) {
	conn, _ := net.Pipe()

	s, err := NewSession(withTestConn(conn))
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Unpexected error when closing Session (first close): %v", err)
	}

	if err := s.Close(); err != nil {
		t.Fatalf("Unpexected error when closing Session (second close): %v", err)
	}
}
