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
	"bytes"
	"encoding/binary"
	"net"
	"reflect"
	"testing"
)

func TestTransportSend(t *testing.T) {
	client, srvr := net.Pipe()
	defer client.Close()
	defer srvr.Close()

	tr := &transport{
		conn: client,
	}

	done := make(chan struct{}, 1)

	// Send packet and ensure that what is read matches the gold bytes
	go func() {
		defer close(done)

		// Read the header to get the packet length...
		b := make([]byte, headerLength)

		_, err := srvr.Read(b)
		if err != nil {
			t.Errorf("Unexpected error reading bytes: %v", err)
		}

		length := binary.BigEndian.Uint32(b)

		if want := len(goldNamedPacketBytes); length != uint32(want) {
			t.Errorf("Unexpected packet length: got %d, expected: %d", length, want)
		}

		// Read the packet data...
		b = make([]byte, length)

		_, err = srvr.Read(b)
		if err != nil {
			t.Errorf("Unexpected error reading bytes: %v", err)
		}

		if !bytes.Equal(b, goldNamedPacketBytes) {
			t.Errorf("Received byte stream does not equal gold bytes.\nExpected: %v\nReceived: %v", goldUnnamedPacketBytes, b)
		}
	}()

	err := tr.send(goldNamedPacket)
	if err != nil {
		t.Errorf("Unexpected error sending packet: %v", err)
	}

	<-done
}

func TestTransportRecv(t *testing.T) {
	client, srvr := net.Pipe()
	defer client.Close()
	defer srvr.Close()

	tr := &transport{
		conn: client,
	}

	done := make(chan struct{}, 1)

	// Server sends bytes, client reads a returns a packet. Ensure that the
	// packet is goldNamedPacket
	go func() {
		defer close(done)

		p, err := tr.recv()
		if err != nil {
			t.Errorf("Unexpected error receiving packet: %v", err)
		}

		if !reflect.DeepEqual(p, goldNamedPacket) {
			t.Errorf("Received packet does not equal gold packet.\nExpected: %v\n Received: %v", goldNamedPacket, p)
		}
	}()

	// Make a buffer big enough for the data and the header.
	b := make([]byte, headerLength+len(goldNamedPacketBytes))

	binary.BigEndian.PutUint32(b[:headerLength], uint32(len(goldNamedPacketBytes)))

	copy(b[headerLength:], goldNamedPacketBytes)

	_, err := srvr.Write(b)
	if err != nil {
		t.Errorf("Unexpected error sending bytes: %v", err)
	}

	<-done
}
