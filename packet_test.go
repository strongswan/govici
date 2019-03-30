//
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
//

package vici

import (
	"bytes"
	"reflect"
	"testing"
)

var (
	goldNamedPacket = &packet{
		ptype: pktCmdRequest,
		name:  "install",
		msg: &Message{
			keys: []string{"child", "ike"},
			data: map[string]interface{}{
				"child": "test-CHILD_SA",
				"ike":   "test-IKE_SA",
			},
		},
	}

	goldNamedPacketBytes = []byte{
		// Packet type
		0,
		// Length of "install"
		7,
		// "install" in bytes
		105, 110, 115, 116, 97, 108, 108,
		// Encoded message bytes
		3, 5, 99, 104, 105, 108, 100, 0, 13, 116, 101, 115, 116,
		45, 67, 72, 73, 76, 68, 95, 83, 65, 3, 3, 105, 107, 101,
		0, 11, 116, 101, 115, 116, 45, 73, 75, 69, 95, 83, 65,
	}

	goldUnnamedPacket = &packet{
		ptype: pktCmdResponse,
		msg: &Message{
			keys: []string{"success", "errmsg"},
			data: map[string]interface{}{
				"success": "no",
				"errmsg":  "failed to install CHILD_SA",
			},
		},
	}

	goldUnnamedPacketBytes = []byte{
		// Packet type
		1,
		// Encoded message bytes
		3, 7, 115, 117, 99, 99, 101, 115, 115, 0, 2, 110, 111, 3, 6,
		101, 114, 114, 109, 115, 103, 0, 26, 102, 97, 105, 108, 101,
		100, 32, 116, 111, 32, 105, 110, 115, 116, 97, 108, 108, 32,
		67, 72, 73, 76, 68, 95, 83, 65,
	}
)

func TestPacketParse(t *testing.T) {
	p := &packet{}

	err := p.parse(goldNamedPacketBytes)
	if err != nil {
		t.Errorf("Error parsing packet: %v", err)
	}

	if !reflect.DeepEqual(p, goldNamedPacket) {
		t.Errorf("Parsed named packet does not equal gold packet.\nExpected: %v\nReceived: %v", goldNamedPacket, p)
	}

	p = &packet{}

	err = p.parse(goldUnnamedPacketBytes)
	if err != nil {
		t.Errorf("Error parsing packet: %v", err)
	}

	if !reflect.DeepEqual(p, goldUnnamedPacket) {
		t.Errorf("Parsed unnamed packet does not equal gold packet.\nExpected: %v\nReceived: %v", goldUnnamedPacket, p)
	}
}

func TestPacketBytes(t *testing.T) {
	b, err := goldNamedPacket.bytes()
	if err != nil {
		t.Errorf("Unexpected error getting packet bytes: %v", err)
	}

	if !bytes.Equal(b, goldNamedPacketBytes) {
		t.Errorf("Encoded packet does not equal gold bytes.\nExpected: %v\nReceived: %v", goldNamedPacketBytes, b)
	}

	b, err = goldUnnamedPacket.bytes()
	if err != nil {
		t.Errorf("Unexpected error getting packet bytes: %v", err)
	}

	if !bytes.Equal(b, goldUnnamedPacketBytes) {
		t.Errorf("Encoded packet does not equal gold bytes.\nExpected: %v\nReceived: %v", goldUnnamedPacketBytes, b)
	}
}
