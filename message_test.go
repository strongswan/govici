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

// Package vici implements a strongSwan vici protocol client
package vici

import (
	"bytes"
	"reflect"
	"testing"
)

var (
	// Gold message
	goldMessage = &message{
		data: map[string]interface{}{
			"key1": "value1",
			"section1": map[string]interface{}{
				"sub-section": map[string]interface{}{
					"key2": "value2",
				},
				"list1": []string{"item1", "item2"},
			},
		},
	}

	// Expected byte stream from encoding testMessage
	goldMessageBytes = []byte{
		// key1 = value1
		3, 4, 'k', 'e', 'y', '1', 0, 6, 'v', 'a', 'l', 'u', 'e', '1',
		// section1
		1, 8, 's', 'e', 'c', 't', 'i', 'o', 'n', '1',
		// sub-section
		1, 11, 's', 'u', 'b', '-', 's', 'e', 'c', 't', 'i', 'o', 'n',
		// key2 = value2
		3, 4, 'k', 'e', 'y', '2', 0, 6, 'v', 'a', 'l', 'u', 'e', '2',
		// sub-section end
		2,
		// list1
		4, 5, 'l', 'i', 's', 't', '1',
		// item1
		5, 0, 5, 'i', 't', 'e', 'm', '1',
		// item2
		5, 0, 5, 'i', 't', 'e', 'm', '2',
		// list1 end
		6,
		// section1 end
		2,
	}
)

func TestMessageEncode(t *testing.T) {
	b, err := goldMessage.encode()
	if err != nil {
		t.Errorf("Error encoding test message: %v", err)
	}

	if !bytes.Equal(b, goldMessageBytes) {
		t.Errorf("Encoded message does not equal gold bytes.\nExpected: %v\nReceived: %v", goldMessageBytes, b)
	}
}

func TestMessageDecode(t *testing.T) {
	m := newMessage()
	err := m.decode(goldMessageBytes)
	if err != nil {
		t.Errorf("Error decoding test bytes: %v", err)
	}

	if !reflect.DeepEqual(m.data, goldMessage.data) {
		t.Errorf("Decoded message does not equal gold message.\nExpected: %v\nReceived: %v", goldMessage.data, m.data)
	}
}
