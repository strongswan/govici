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
	"reflect"
	"testing"
)

var (
	// Gold message
	goldMessage = &Message{
		keys: []string{"key1", "section1"},
		data: map[string]interface{}{
			"key1": "value1",
			// Section is another message
			"section1": &Message{
				keys: []string{"sub-section", "list1"},
				data: map[string]interface{}{
					// Sub-section is a another message
					"sub-section": &Message{
						keys: []string{"key2"},
						data: map[string]interface{}{
							"key2": "value2",
						},
					},
					"list1": []string{"item1", "item2"},
				},
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

	goldUnmarshaled = testMessage{
		Key:  "value",
		List: []string{"item1", "item2"},
		Message: &Message{
			keys: []string{"key1"},
			data: map[string]interface{}{
				"key1": "value1",
			},
		},
		Section1: testSection{Key: "key2"},
		Section2: &testSection{List: []string{"item3", "item4"}},
	}

	goldMarshaled = &Message{
		keys: []string{"key", "list", "message", "section1", "section2"},
		data: map[string]interface{}{
			"key":  "value",
			"list": []string{"item1", "item2"},
			"message": &Message{
				keys: []string{"key1"},
				data: map[string]interface{}{
					"key1": "value1",
				},
			},
			"section1": &Message{
				keys: []string{"key"},
				data: map[string]interface{}{
					"key": "key2",
				},
			},
			"section2": &Message{
				keys: []string{"list"},
				data: map[string]interface{}{
					"list": []string{"item3", "item4"},
				},
			},
		},
	}
)

type testMessage struct {
	Key      string       `vici:"key"`
	Empty    string       `vici:"empty"`
	List     []string     `vici:"list"`
	Message  *Message     `vici:"message"`
	Section1 testSection  `vici:"section1"`
	Section2 *testSection `vici:"section2"`
	Skip     string       `vici:"-"`

	NotTagged  string
	unexported string
}

type testSection struct {
	Key  string   `vici:"key"`
	List []string `vici:"list"`
}

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
	m := NewMessage()
	err := m.decode(goldMessageBytes)
	if err != nil {
		t.Errorf("Error decoding test bytes: %v", err)
	}

	if !reflect.DeepEqual(m.data, goldMessage.data) {
		t.Errorf("Decoded message does not equal gold message.\nExpected: %v\nReceived: %v", goldMessage.data, m.data)
	}
}

func TestMarshalMessage(t *testing.T) {
	m, err := MarshalMessage(goldUnmarshaled)
	if err != nil {
		t.Errorf("Unexpected error marshaling: %v", err)
	}

	if !reflect.DeepEqual(m, goldMarshaled) {
		t.Errorf("Marshaled message does not equal gold marshaled message.\nExpected: %v\nReceived: %v", goldMarshaled, m)
	}
}

func TestUnmarshalMessage(t *testing.T) {
	tm := &testMessage{
		Message:  NewMessage(),
		Section2: &testSection{},
	}
	err := UnmarshalMessage(goldMarshaled, tm)
	if err != nil {
		t.Errorf("Unexpected error unmarshaling: %v", err)
	}

	if !reflect.DeepEqual(*tm, goldUnmarshaled) {
		t.Errorf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled, *tm)
	}
}
