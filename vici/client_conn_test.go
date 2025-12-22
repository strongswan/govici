// Copyright (C) 2025 Nick Rosbrook
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
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"testing/synctest"
	"time"
)

type testServer struct {
	conn net.Conn

	sync.Mutex
	data map[string]any
}

func newTestServer(conn net.Conn) *testServer {
	ts := &testServer{
		conn: conn,
		data: make(map[string]any),
	}

	return ts
}

func (ts *testServer) commandHandlerOK(_ *Message) (*Message, error) {
	resp := &Message{
		header: &header{
			ptype: pktCmdResponse,
		},
	}

	return resp, nil
}

func (ts *testServer) commandHandlerError(_ *Message) (*Message, error) {
	resp := NewMessage()
	resp.header = &header{
		ptype: pktCmdResponse,
	}

	if err := resp.Set("success", "no"); err != nil {
		return nil, err
	}
	if err := resp.Set("errmsg", "cmd-err: failure"); err != nil {
		return nil, err
	}

	return resp, nil
}

func (ts *testServer) commandHandlerTimeout(_ *Message) (*Message, error) {
	resp := &Message{
		header: &header{
			ptype: pktCmdResponse,
		},
	}

	time.Sleep(3 * time.Second)

	return resp, nil
}

func (ts *testServer) commandHandlerTimeoutLong(_ *Message) (*Message, error) {
	resp := &Message{
		header: &header{
			ptype: pktCmdResponse,
		},
	}

	time.Sleep(10 * time.Second)

	return resp, nil
}

func (ts *testServer) commandHandlerNoResponse(_ *Message) (*Message, error) {
	return nil, syscall.ENODATA
}

func (ts *testServer) commandHandlerStrcat(in *Message) (*Message, error) {
	resp := NewMessage()
	resp.header = &header{
		ptype: pktCmdResponse,
	}

	a, ok := in.Get("a").(string)
	if !ok {
		return nil, errors.New("malformed message")
	}

	b, ok := in.Get("b").(string)
	if !ok {
		return nil, errors.New("malformed message")
	}

	if err := resp.Set("c", a+b); err != nil {
		return nil, err
	}

	return resp, nil
}

func (ts *testServer) commandHandlerStream(_ *Message) (*Message, error) {
	ts.Lock()
	defer ts.Unlock()

	v, ok := ts.data["cmd-stream-signal"]

	if !ok {
		return nil, nil
	}

	wait := v.(chan struct{})

	<-wait // Read to signal that the command was called.
	<-wait // Wait until the events are done.

	resp := NewMessage()
	resp.header = &header{
		ptype: pktCmdResponse,
	}
	if err := resp.Set("done", "yes"); err != nil {
		return nil, err
	}

	return resp, nil
}

func (ts *testServer) commandHandler(p *Message) (*Message, error) {
	switch p.header.name {
	case "cmd-ok":
		return ts.commandHandlerOK(p)
	case "cmd-err":
		return ts.commandHandlerError(p)
	case "cmd-timeout":
		return ts.commandHandlerTimeout(p)
	case "cmd-timeout-long":
		return ts.commandHandlerTimeoutLong(p)
	case "cmd-strcat":
		return ts.commandHandlerStrcat(p)
	case "cmd-stream":
		return ts.commandHandlerStream(p)
	case "cmd-no-response":
		return ts.commandHandlerNoResponse(p)
	default:
		return nil, nil
	}
}

func (ts *testServer) eventRegisterHandlerConfirm(_ *Message) (*Message, error) {
	resp := &Message{
		header: &header{
			ptype: pktEventConfirm,
		},
	}

	return resp, nil
}

func (ts *testServer) eventRegisterHandlerSimple(p *Message) (*Message, error) {
	resp := &Message{
		header: &header{
			ptype: pktEventConfirm,
		},
	}

	if p.header.ptype == pktEventUnregister {
		return resp, nil
	}

	go func() {
		for i := 0; i < 3; i++ {
			time.Sleep(time.Second)

			ev := NewMessage()
			ev.header = &header{
				ptype: pktEvent,
				name:  "event-simple",
			}
			if err := ev.Set("index", i); err != nil {
				panic(err)
			}

			if err := sendmsg(ts.conn, ev); err != nil {
				panic(err)
			}
		}
	}()

	return resp, nil
}

func (ts *testServer) eventRegisterHandlerStream(p *Message) (*Message, error) {
	ts.Lock()
	defer ts.Unlock()

	resp := &Message{
		header: &header{
			ptype: pktEventConfirm,
		},
	}

	if p.header.ptype == pktEventUnregister {
		delete(ts.data, "cmd-stream-signal")

		return resp, nil
	}

	signal := make(chan struct{})
	ts.data["cmd-stream-signal"] = signal

	go func() {
		// Block until the command is called.
		signal <- struct{}{}

		for i := 0; i < 3; i++ {
			ev := NewMessage()
			ev.header = &header{
				ptype: pktEvent,
				name:  "event-stream",
			}
			if err := ev.Set("index", i); err != nil {
				panic(err)
			}

			if err := sendmsg(ts.conn, ev); err != nil {
				panic(err)
			}
		}

		close(signal)
	}()

	return resp, nil
}

func (ts *testServer) handleEventRegistration(p *Message) (*Message, error) {
	switch p.header.name {
	case "event-confirm":
		return ts.eventRegisterHandlerConfirm(p)
	case "event-simple":
		return ts.eventRegisterHandlerSimple(p)
	case "event-stream":
		return ts.eventRegisterHandlerStream(p)
	default:
		resp := &Message{
			header: &header{
				ptype: pktEventUnknown,
			},
		}

		return resp, nil
	}
}

func (ts *testServer) serve() {
	for {
		var (
			err  error
			resp *Message
		)

		p, err := readmsg(ts.conn)
		if err != nil {
			if errors.Is(err, io.ErrClosedPipe) {
				return
			}
			panic(err)
		}

		go func() {
			switch p.header.ptype {
			case pktCmdRequest:
				resp, err = ts.commandHandler(p)
				if errors.Is(err, syscall.ENODATA) {
					// Specical case that means send no response.
					return
				}
				if err != nil {
					panic(err)
				}

			case pktEventRegister, pktEventUnregister:
				resp, err = ts.handleEventRegistration(p)
				if err != nil {
					panic(err)
				}
			default:
				return
			}

			if resp == nil {
				resp = &Message{
					header: &header{},
				}

				switch p.header.ptype {
				case pktCmdRequest:
					resp.header.ptype = pktCmdUnknown
				case pktEventRegister, pktEventUnregister:
					resp.header.ptype = pktEventUnknown
				}
			}

			if err := sendmsg(ts.conn, resp); err != nil {
				panic(err)
			}
		}()
	}
}

func newTestClientServer() (*clientConn, *testServer) {
	client, server := net.Pipe()

	cc := newClientConn(client)
	ts := newTestServer(server)

	go cc.listen()
	go ts.serve()

	return cc, ts
}

func TestClientConnWrite(t *testing.T) {
	var (
		wg sync.WaitGroup
	)

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()

		cc := newClientConn(client)

		// Send packet and ensure that what is read matches the gold bytes
		err := cc.write(context.Background(), goldNamedPacket)
		if err != nil {
			t.Errorf("Unexpected error sending packet: %v", err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Read the header to get the packet length...
		b := make([]byte, 4 /* header length */)

		_, err := server.Read(b)
		if err != nil {
			t.Errorf("Unexpected error reading bytes: %v", err)
			return
		}

		length := binary.BigEndian.Uint32(b)

		// #nosec G115
		if want := len(goldNamedPacketBytes); length != uint32(want) {
			t.Errorf("Unexpected packet length: got %d, expected: %d", length, want)
			return
		}

		// Read the packet data...
		b = make([]byte, length)

		_, err = server.Read(b)
		if err != nil {
			t.Errorf("Unexpected error reading bytes: %v", err)
			return
		}

		if !bytes.Equal(b, goldNamedPacketBytes) {
			t.Errorf("Received byte stream does not equal gold bytes.\nExpected: %v\nReceived: %v", goldUnnamedPacketBytes, b)
			return
		}
	}()

	wg.Wait()
}

func TestClientConnRead(t *testing.T) {
	var (
		wg sync.WaitGroup
	)

	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	wg.Add(1)
	go func() {
		defer wg.Done()

		// Make a buffer big enough for the data and the header.
		raw := make([]byte, 4)
		binary.BigEndian.PutUint32(raw, uint32(len(goldUnnamedPacketBytes))) // #nosec G115
		buf := bytes.NewBuffer(raw)

		if _, err := buf.Write(goldUnnamedPacketBytes); err != nil {
			t.Errorf("Unexpected error writing packet: %v", err)
			return
		}

		if _, err := server.Write(buf.Bytes()); err != nil {
			t.Errorf("Unexpected error sending bytes: %v", err)
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		cc := newClientConn(client)

		p, err := readmsg(cc.conn)
		if err != nil {
			t.Errorf("Unexpected error receiving packet: %v", err)
			return
		}

		// Server sends bytes, client reads a returns a packet. Ensure that the
		// packet is goldNamedPacket
		if !reflect.DeepEqual(p, goldUnnamedPacket) {
			t.Errorf("Received packet does not equal gold packet.\nExpected: %v\n Received: %v", goldUnnamedPacket, p)
			return
		}
	}()

	wg.Wait()
}

func TestClientConnWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cc, ts := newTestClientServer()
		defer cc.conn.Close()
		defer ts.conn.Close()

		in := NewMessage()
		in.header = &header{
			ptype: pktCmdRequest,
			name:  "cmd-test",
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		synctest.Wait()
		if err := cc.write(ctx, in); !errors.Is(err, context.Canceled) {
			t.Fatalf("Expected cancel on write, but got %v", err)
		}

		ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		in.header.name = "cmd-timeout"
		if err := cc.write(ctx, in); err != nil {
			t.Fatalf("Failed to write %s: %v", in.header.name, err)
		}

		synctest.Wait()
		if _, err := cc.wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Expected timeout on wait, but got %v", err)
		}

		in.header.name = "cmd-ok"
		if err := cc.write(context.Background(), in); err != nil {
			t.Fatalf("Failed to write %s: %v", in.header.name, err)
		}

		m, err := cc.wait(context.Background())
		if err != nil {
			t.Fatalf("Failed to wait: %v", err)
		}

		if m.header.seq != 2 {
			t.Fatalf("Received the wrong message!")
		}
	})
}

func TestClientConnWaitNoResponse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cc, ts := newTestClientServer()
		defer cc.conn.Close()
		defer ts.conn.Close()

		in := NewMessage()
		in.header = &header{
			ptype: pktCmdRequest,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		in.header.name = "cmd-no-response"
		if err := cc.write(ctx, in); err != nil {
			t.Fatalf("Failed to write %s: %v", in.header.name, err)
		}

		synctest.Wait()
		if _, err := cc.wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Expected deadline exceeded on wait, but got %v", err)
		}

		// Without a context deadline, we should get a timeout error from
		// the default read deadline.
		in.header.name = "cmd-no-response"
		if err := cc.write(context.Background(), in); err != nil {
			t.Fatalf("Failed to write %s: %v", in.header.name, err)
		}

		synctest.Wait()
		if _, err := cc.wait(context.Background()); !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("Expected timeout on wait, but got %v", err)
		}
	})
}

func TestClientConnWaitDelayedResponse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cc, ts := newTestClientServer()
		defer cc.conn.Close()
		defer ts.conn.Close()

		in := NewMessage()
		in.header = &header{
			ptype: pktCmdRequest,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		in.header.name = "cmd-timeout-long"
		if err := cc.write(ctx, in); err != nil {
			t.Fatalf("Failed to write %s: %v", in.header.name, err)
		}

		synctest.Wait()
		if _, err := cc.wait(ctx); err != nil {
			t.Fatalf("Expected success with long context deadline, but got %v", err)
		}
	})
}

func TestClientConnCall(t *testing.T) {
	cc, ts := newTestClientServer()
	defer cc.conn.Close()
	defer ts.conn.Close()

	if _, err := cc.call(context.Background(), "cmd-ok", nil); err != nil {
		t.Fatalf("Unexpected failure: %v", err)
	}

	if _, err := cc.call(context.Background(), "cmd-unknown", nil); !errors.Is(err, errUnexpectedResponse) {
		t.Fatalf("Expected to receive %v, but got %v", errUnexpectedResponse, err)
	}

	if _, err := cc.call(context.Background(), "cmd-err", nil); !errors.Is(err, errCommandFailed) {
		t.Fatalf("Expected to receive %v, but got %v", errCommandFailed, err)
	}

	in := NewMessage()
	if err := in.Set("a", "test"); err != nil {
		t.Fatal(err)
	}
	if err := in.Set("b", "123"); err != nil {
		t.Fatal(err)
	}

	out, err := cc.call(context.Background(), "cmd-strcat", in)
	if err != nil {
		t.Fatalf("Unexpected failure: %v", err)
	}

	c, ok := out.Get("c").(string)
	if !ok || c != "test123" {
		t.Fatalf("Expected field c=test123 in %s", out)
	}
}

func TestClientConnSubscribe(t *testing.T) {
	cc, ts := newTestClientServer()
	defer cc.conn.Close()
	defer ts.conn.Close()

	if err := cc.subscribe(context.Background(), "event-confirm"); err != nil {
		t.Fatalf("Unexpected failure: %v", err)
	}

	if err := cc.subscribe(context.Background(), "event-unknown"); !errors.Is(err, errEventUnknown) {
		t.Fatalf("Expected to receive %v, but got %v", errEventUnknown, err)
	}

	if err := cc.unsubscribe(context.Background(), "event-confirm"); err != nil {
		t.Fatalf("Unexpected failure: %v", err)
	}
}

func TestClientConnNotify(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cc, ts := newTestClientServer()
		defer cc.conn.Close()
		defer ts.conn.Close()

		ec := make(chan Event, 3)
		cc.notify(ec)
		defer cc.unnotify(ec)

		if err := cc.subscribe(context.Background(), "event-simple"); err != nil {
			t.Fatalf("Unexpected failure: %v", err)
		}

		for i := 0; i < 3; i++ {
			synctest.Wait()

			select {
			case ev, ok := <-ec:
				if !ok {
					t.Fatalf("Unexpected chan closure")
				}
				if ev.Name != "event-simple" {
					t.Fatalf("Received unexpected message %s", ev.Name)
				}
				if v, ok := ev.Message.Get("index").(string); !ok || v != strconv.Itoa(i) {
					t.Fatalf("Unexpected message contents: %s", ev.Message)
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("Did not receive event %d/3!", i+1)
			}
		}

		if err := cc.unsubscribe(context.Background(), "event-simple"); err != nil {
			t.Fatalf("Unexpected failure: %v", err)
		}
	})
}

func TestClientConnStream(t *testing.T) {
	cc, ts := newTestClientServer()
	defer cc.conn.Close()
	defer ts.conn.Close()

	// Ensure that internal events aren't leaked to subscribers.
	ec := make(chan Event, 4)
	cc.notify(ec)
	defer cc.unnotify(ec)

	i := 0
	for m, err := range cc.stream(context.Background(), "cmd-stream", "event-stream", nil) {
		if err != nil {
			t.Fatalf("Unexpected failure: %v", err)
		}

		if m.Err() != nil {
			t.Fatalf("Unexpected failure: %v", err)
		}

		if v, ok := m.Get("index").(string); ok {
			if v != strconv.Itoa(i) {
				t.Fatalf("Unexpected message contents: %s", m)
			}
			i++
		} else if _, ok := m.Get("done").(string); !ok {
			t.Fatalf("Unexpected non-event message contents: %s", m)
		}
	}

	select {
	case ev := <-ec:
		t.Fatalf("Should not have received internal event, but got %s", ev.Name)
	default:
	}
}
