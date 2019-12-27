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
	"sort"
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

	goldUnmarshaledMap = map[string]interface{}{
		"key":      goldUnmarshaled.Key,
		"list":     goldUnmarshaled.List,
		"message":  goldUnmarshaled.Message,
		"section1": goldUnmarshaled.Section1,
		"section2": goldUnmarshaled.Section2,
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

func TestMarshalMessageMap(t *testing.T) {
	m, err := MarshalMessage(goldUnmarshaledMap)
	if err != nil {
		t.Errorf("Unexpected error marshaling: %v", err)
	}

	// Map keys are unordered, so we need to compare differently

	marshaledKeys := append(goldMarshaled.keys[:0:0], goldMarshaled.keys...)
	sort.Strings(m.keys)
	sort.Strings(marshaledKeys)

	if !reflect.DeepEqual(m.keys, marshaledKeys) {
		t.Errorf("Marshaled message does not equal gold marshaled message keys.\nExpected: %v\nReceived: %v", marshaledKeys, m.data)
	}

	if !reflect.DeepEqual(m.data, goldMarshaled.data) {
		t.Errorf("Marshaled message does not equal gold marshaled message data.\nExpected: %v\nReceived: %v", goldMarshaled.data, m.data)
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

func TestUnmarshalMessageMapSimple(t *testing.T) {

	marshaled := &Message{
		keys: []string{"one", "two"},
		data: map[string]interface{}{
			"one": "1",
			"two": "2",
		},
	}

	tm := map[string]string{}

	err := UnmarshalMessage(marshaled, tm)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling ot map: %v", err)
	}

	one, ok := tm["one"]
	if !ok {
		t.Fatalf("Unmarshaled message does not contain 'one'")
	}
	if one != "1" {
		t.Errorf("Value of 'one' is incorrect.\nExpected: 1\nReceived: %s", one)
	}

	two, ok := tm["two"]
	if !ok {
		t.Fatalf("Unmarshaled message does not contain 'two'")
	}
	if two != "2" {
		t.Errorf("Value of 'two' is incorrect.\nExpected: 2\nReceived: %s", two)
	}

}

func TestUnmarshalMessageMapPointers(t *testing.T) {
	marshaled := &Message{
		keys: []string{"section1", "section2"},
		data: map[string]interface{}{
			"section1": goldMarshaled.data["section1"],
			"section2": goldMarshaled.data["section2"],
		},
	}

	tm := map[string]*testSection{}

	err := UnmarshalMessage(marshaled, tm)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling to map: %v", err)
	}

	if !reflect.DeepEqual(tm["section1"], &goldUnmarshaled.Section1) {
		t.Errorf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", &goldUnmarshaled.Section1, tm["section1"])
	}

	if !reflect.DeepEqual(tm["section2"], goldUnmarshaled.Section2) {
		t.Errorf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled.Section2, tm["section2"])
	}
}

func TestUnmarshalMessageMapNotPointers(t *testing.T) {
	marshaled := &Message{
		keys: []string{"section1", "section2"},
		data: map[string]interface{}{
			"section1": goldMarshaled.data["section1"],
			"section2": goldMarshaled.data["section2"]},
	}

	tm := map[string]testSection{}

	err := UnmarshalMessage(marshaled, tm)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling to map: %v", err)
	}

	if !reflect.DeepEqual(tm["section1"], goldUnmarshaled.Section1) {
		t.Errorf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled.Section1, tm["section1"])
	}

	if !reflect.DeepEqual(tm["section2"], *goldUnmarshaled.Section2) {
		t.Errorf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", *goldUnmarshaled.Section2, tm["section2"])
	}
}

func TestUnmarshalMessageNotPointer(t *testing.T) {
	tm := testMessage{
		Message:  NewMessage(),
		Section2: &testSection{},
	}

	err := UnmarshalMessage(goldMarshaled, tm)
	if err == nil {
		t.Errorf("Expected error when unmarshaling to non-pointer struct")
	}
}

func TestUnmarshalMessageToNilPointer(t *testing.T) {
	var tm *testMessage

	err := UnmarshalMessage(goldMarshaled, tm)
	if err == nil {
		t.Errorf("Expected error when unmarshaling to nil pointer")
	}
}

func TestUnmarshalMessageNestedNilPtr(t *testing.T) {
	tm := &testMessage{
		Message: NewMessage(),
	}

	err := UnmarshalMessage(goldMarshaled, tm)
	if err != nil {
		t.Errorf("Unexpected error unmarshaling: %v", err)
	}

	if !reflect.DeepEqual(*tm, goldUnmarshaled) {
		t.Errorf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled, *tm)
	}
}

func TestMessageGet(t *testing.T) {
	v := goldMessage.Get("key1")
	if value, ok := v.(string); !ok {
		t.Errorf("Expected %v to be string: received %T", value, value)
	} else if value != "value1" {
		t.Errorf("Expected 'key1' to be 'value1': received %v", value)
	}

	v = goldMessage.Get("invalid")
	if v != nil {
		t.Errorf("Expected nil for Get on non-existent key: received %v", v)
	}
}

func TestMessageSet(t *testing.T) {
	// Test that all supported types can be set.
	valid := []interface{}{
		// type: string
		"value",

		// type: []string
		[]string{"item1", "item2"},

		// type: *Message
		NewMessage(),

		// type: int
		0,

		// type: bool
		true,

		// type: map
		map[string]interface{}{
			"key1": "value1",
			"key2": []string{"item1", "item2"},
		},

		// type: struct (must be valid as per MarshalMessage)
		struct {
			Name string `vici:"name"`
		}{
			Name: "value1",
		},
	}

	m := NewMessage()

	for _, v := range valid {
		if err := m.Set("test", v); err != nil {
			t.Fatalf("unexpected error setting supported type '%T': %v", v, err)
		}
	}
}

func TestMessageSetTypeConversion(t *testing.T) {
	type conversion struct {
		in  interface{}
		out interface{}
	}

	conversions := []conversion{
		{"string", "string"},
		{[]string{"item1", "item2"}, []string{"item1", "item2"}},
		{3, "3"},
		{true, "yes"},
		{false, "no"},
	}

	m := NewMessage()

	for _, c := range conversions {
		if err := m.Set("test", c.in); err != nil {
			t.Fatalf("unexpected error setting supported type '%T': %v", c.in, err)
		}

		if !reflect.DeepEqual(c.out, m.Get("test")) {
			t.Fatalf("got incorrect conversion '%T'\nexpected: %v\n got: %v", c.in, c.out, m.Get("test"))
		}
	}
}

func TestMessageUniqueKeys(t *testing.T) {
	m := NewMessage()

	err := m.Set("key1", "firstValue")
	if err != nil {
		t.Fatalf("Unexpected error setting string in message: %v", err)
	}

	// Make sure that keys are not duplicated
	err = m.Set("key1", "newValue")
	if err != nil {
		t.Fatalf("Unexpected error setting string in message: %v", err)
	}

	if v := m.Get("key1"); v.(string) != "newValue" {
		t.Fatalf("Expected old value of 'key1' to be overwritten: key1=%v", v)
	}

	indices := make([]int, 0)
	for i, v := range m.Keys() {
		if v == "key1" {
			indices = append(indices, i)
		}
	}

	if len(indices) != 1 {
		t.Fatalf("Expected unique message keys: found %v instances of 'key1'", len(indices))
	}
}
