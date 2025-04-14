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
	"fmt"
	"reflect"
	"sort"
	"testing"
)

var (
	goldNamedPacket = &Message{
		header: &header{
			ptype: pktCmdRequest,
			name:  "install",
		},
		keys: []string{"child", "ike"},
		data: map[string]any{
			"child": "test-CHILD_SA",
			"ike":   "test-IKE_SA",
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

	goldUnnamedPacket = &Message{
		header: &header{
			ptype: pktCmdResponse,
		},
		keys: []string{"success", "errmsg"},
		data: map[string]any{
			"success": "no",
			"errmsg":  "failed to install CHILD_SA",
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

	// Gold message
	goldMessage = &Message{
		header: &header{
			ptype: pktCmdResponse,
		},
		keys: []string{"key1", "section1"},
		data: map[string]any{
			"key1": "value1",
			// Section is another message
			"section1": &Message{
				keys: []string{"sub-section", "list1"},
				data: map[string]any{
					// Sub-section is a another message
					"sub-section": &Message{
						keys: []string{"key2"},
						data: map[string]any{
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
		// pktCmdResponse
		1,
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
			data: map[string]any{
				"key1": "value1",
			},
		},
		Section1: testSection{Key: "key2"},
		Section2: &testSection{List: []string{"item3", "item4"}},
	}

	goldUnmarshaledMap = map[string]any{
		"key":      goldUnmarshaled.Key,
		"list":     goldUnmarshaled.List,
		"message":  goldUnmarshaled.Message,
		"section1": goldUnmarshaled.Section1,
		"section2": goldUnmarshaled.Section2,
	}

	goldMarshaled = &Message{
		keys: []string{"key", "list", "message", "section1", "section2"},
		data: map[string]any{
			"key":  "value",
			"list": []string{"item1", "item2"},
			"message": &Message{
				keys: []string{"key1"},
				data: map[string]any{
					"key1": "value1",
				},
			},
			"section1": &Message{
				keys: []string{"key"},
				data: map[string]any{
					"key": "key2",
				},
			},
			"section2": &Message{
				keys: []string{"list"},
				data: map[string]any{
					"list": []string{"item3", "item4"},
				},
			},
		},
	}
)

func TestPacketParse(t *testing.T) {
	m := NewMessage()

	if err := m.decode(goldNamedPacketBytes); err != nil {
		t.Fatalf("Error parsing packet: %v", err)
	}

	if !reflect.DeepEqual(m, goldNamedPacket) {
		t.Fatalf("Parsed named packet does not equal gold packet.\nExpected: %v\nReceived: %v", goldNamedPacket, m)
	}

	m = NewMessage()

	if err := m.decode(goldUnnamedPacketBytes); err != nil {
		t.Fatalf("Error parsing packet: %v", err)
	}

	if !reflect.DeepEqual(m, goldUnnamedPacket) {
		t.Fatalf("Parsed unnamed packet does not equal gold packet.\nExpected: %v\nReceived: %v", goldUnnamedPacket, m)
	}
}

func TestPacketBytes(t *testing.T) {
	b, err := goldNamedPacket.encode()
	if err != nil {
		t.Fatalf("Unexpected error getting packet bytes: %v", err)
	}

	if !bytes.Equal(b, goldNamedPacketBytes) {
		t.Fatalf("Encoded packet does not equal gold bytes.\nExpected: %v\nReceived: %v", goldNamedPacketBytes, b)
	}

	b, err = goldUnnamedPacket.encode()
	if err != nil {
		t.Fatalf("Unexpected error getting packet bytes: %v", err)
	}

	if !bytes.Equal(b, goldUnnamedPacketBytes) {
		t.Fatalf("Encoded packet does not equal gold bytes.\nExpected: %v\nReceived: %v", goldUnnamedPacketBytes, b)
	}
}

func TestPacketTooLong(t *testing.T) {
	tooLong := make([]byte, 256)

	for i := range tooLong {
		tooLong[i] = 'a'
	}

	m := &Message{
		header: &header{
			ptype: pktCmdRequest,
			name:  string(tooLong),
		},
	}

	_, err := m.encode()
	if err == nil {
		t.Fatalf("Expected packet-too-long error due to %s", m.header.name)
	}
}

type testMessage struct {
	Key      string       `vici:"key"`
	Empty    string       `vici:"empty"`
	List     []string     `vici:"list"`
	Message  *Message     `vici:"message"`
	Section1 testSection  `vici:"section1"`
	Section2 *testSection `vici:"section2"`
	Skip     string       `vici:"-"`

	NotTagged string
}

type testSection struct {
	Key  string   `vici:"key"`
	List []string `vici:"list"`
}

func TestMessageEncode(t *testing.T) {
	b, err := goldMessage.encode()
	if err != nil {
		t.Fatalf("Error encoding test message: %v", err)
	}

	if !bytes.Equal(b, goldMessageBytes) {
		t.Fatalf("Encoded message does not equal gold bytes.\nExpected: %v\nReceived: %v", goldMessageBytes, b)
	}
}

func TestMessageDecode(t *testing.T) {
	m := NewMessage()
	err := m.decode(goldMessageBytes)
	if err != nil {
		t.Fatalf("Error decoding test bytes: %v", err)
	}

	if !reflect.DeepEqual(m.data, goldMessage.data) {
		t.Fatalf("Decoded message does not equal gold message.\nExpected: %v\nReceived: %v", goldMessage.data, m.data)
	}
}

func ExampleMarshalMessage() {
	type child struct {
		LocalTrafficSelectors []string `vici:"local_ts"`
		UpdownScript          string   `vici:"updown"`
		ESPProposals          []string `vici:"esp_proposals"`
	}

	type conn struct {
		LocalAddrs   []string         `vici:"local_addrs"`
		Local        map[string]any   `vici:"local"`
		Remote       map[string]any   `vici:"remote"`
		Children     map[string]child `vici:"children"`
		IKEVersion   uint             `vici:"version"`
		IKEProposals []string         `vici:"proposals"`
	}

	// Create a Message that represents the 'rw' connection from this swanctl.conf:
	// https://www.strongswan.org/testing/testresults/swanctl/rw-cert/moon.swanctl.conf
	rw := &conn{
		LocalAddrs: []string{"192.168.0.1"},
		Local: map[string]any{
			"auth":  "pubkey",
			"certs": []string{"moonCert.pem"},
			"id":    "moon.strongswan.org",
		},
		Remote: map[string]any{
			"auth": "pubkey",
		},
		Children: map[string]child{
			"net": {
				LocalTrafficSelectors: []string{"10.1.0.0/16"},
				UpdownScript:          "/usr/local/libexec/ipsec/_updown iptables",
				ESPProposals:          []string{"aes128gcm128-x25519"},
			},
		},
		IKEVersion:   2,
		IKEProposals: []string{"aes128-sha256-x25519"},
	}

	m, err := MarshalMessage(rw)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(m)
	// Output: {
	//   local_addrs = 192.168.0.1
	//   local {
	//     auth = pubkey
	//     certs = moonCert.pem
	//     id = moon.strongswan.org
	//   }
	//   remote {
	//     auth = pubkey
	//   }
	//   children {
	//     net {
	//       local_ts = 10.1.0.0/16
	//       updown = /usr/local/libexec/ipsec/_updown iptables
	//       esp_proposals = aes128gcm128-x25519
	//     }
	//   }
	//   version = 2
	//   proposals = aes128-sha256-x25519
	// }
}

func ExampleUnmarshalMessage() {
	type child struct {
		LocalTrafficSelectors []string `vici:"local_ts"`
		UpdownScript          string   `vici:"updown"`
		ESPProposals          []string `vici:"esp_proposals"`
	}

	m := NewMessage()

	if err := m.Set("local_ts", []string{"10.1.0.0/16"}); err != nil {
		fmt.Println(err)
		return
	}
	if err := m.Set("esp_proposals", []string{"aes128gcm128-x25519"}); err != nil {
		fmt.Println(err)
		return
	}
	if err := m.Set("updown", "/usr/local/libexec/ipsec/_updown iptables"); err != nil {
		fmt.Println(err)
		return
	}

	var c child

	if err := UnmarshalMessage(m, &c); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v\n", c)
	// Output: {LocalTrafficSelectors:[10.1.0.0/16] UpdownScript:/usr/local/libexec/ipsec/_updown iptables ESPProposals:[aes128gcm128-x25519]}
}

func TestMarshalMessage(t *testing.T) {
	m, err := MarshalMessage(goldUnmarshaled)
	if err != nil {
		t.Fatalf("Unexpected error marshaling: %v", err)
	}

	if !reflect.DeepEqual(m, goldMarshaled) {
		t.Fatalf("Marshaled message does not equal gold marshaled message.\nExpected: %v\nReceived: %v", goldMarshaled, m)
	}
}

func TestMarshalMessageMap(t *testing.T) {
	m, err := MarshalMessage(goldUnmarshaledMap)
	if err != nil {
		t.Fatalf("Unexpected error marshaling: %v", err)
	}

	// Map keys are unordered, so we need to compare differently
	marshaledKeys := make([]string, len(goldMarshaled.keys))
	copy(marshaledKeys, goldMarshaled.keys)
	sort.Strings(m.keys)
	sort.Strings(marshaledKeys)

	if !reflect.DeepEqual(m.keys, marshaledKeys) {
		t.Fatalf("Marshaled message does not equal gold marshaled message keys.\nExpected: %v\nReceived: %v", marshaledKeys, m.data)
	}

	if !reflect.DeepEqual(m.data, goldMarshaled.data) {
		t.Fatalf("Marshaled message does not equal gold marshaled message data.\nExpected: %v\nReceived: %v", goldMarshaled.data, m.data)
	}
}

func TestUnmarshalMessage(t *testing.T) {
	tm := &testMessage{
		Message:  NewMessage(),
		Section2: &testSection{},
	}

	err := UnmarshalMessage(goldMarshaled, tm)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling: %v", err)
	}

	if !reflect.DeepEqual(*tm, goldUnmarshaled) {
		t.Fatalf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled, *tm)
	}
}

func TestUnmarshalMessageMapSimple(t *testing.T) {
	marshaled := &Message{
		keys: []string{"one", "two"},
		data: map[string]any{
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
		t.Fatalf("Value of 'one' is incorrect.\nExpected: 1\nReceived: %s", one)
	}

	two, ok := tm["two"]
	if !ok {
		t.Fatalf("Unmarshaled message does not contain 'two'")
	}
	if two != "2" {
		t.Fatalf("Value of 'two' is incorrect.\nExpected: 2\nReceived: %s", two)
	}
}

func TestUnmarshalMessageMapPointers(t *testing.T) {
	marshaled := &Message{
		keys: []string{"section1", "section2"},
		data: map[string]any{
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
		t.Fatalf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", &goldUnmarshaled.Section1, tm["section1"])
	}

	if !reflect.DeepEqual(tm["section2"], goldUnmarshaled.Section2) {
		t.Fatalf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled.Section2, tm["section2"])
	}
}

func TestUnmarshalMessageMapNotPointers(t *testing.T) {
	marshaled := &Message{
		keys: []string{"section1", "section2"},
		data: map[string]any{
			"section1": goldMarshaled.data["section1"],
			"section2": goldMarshaled.data["section2"]},
	}

	tm := map[string]testSection{}

	err := UnmarshalMessage(marshaled, tm)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling to map: %v", err)
	}

	if !reflect.DeepEqual(tm["section1"], goldUnmarshaled.Section1) {
		t.Fatalf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled.Section1, tm["section1"])
	}

	if !reflect.DeepEqual(tm["section2"], *goldUnmarshaled.Section2) {
		t.Fatalf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", *goldUnmarshaled.Section2, tm["section2"])
	}
}

func TestUnmarshalMessageNotPointer(t *testing.T) {
	tm := testMessage{
		Message:  NewMessage(),
		Section2: &testSection{},
	}

	err := UnmarshalMessage(goldMarshaled, tm)
	if err == nil {
		t.Fatalf("Expected error when unmarshaling to non-pointer struct")
	}
}

func TestUnmarshalMessageToNilPointer(t *testing.T) {
	var tm *testMessage

	err := UnmarshalMessage(goldMarshaled, tm)
	if err == nil {
		t.Fatalf("Expected error when unmarshaling to nil pointer")
	}
}

func TestUnmarshalToNonEmpty(t *testing.T) {
	type data struct {
		A string `vici:"a"`
		B string `vici:"b"`
	}

	d := data{
		A: "testA",
	}

	want := data{
		A: "testA",
		B: "testB",
	}

	m := NewMessage()

	if err := m.Set("b", "testB"); err != nil {
		t.Fatalf("Unable to set message field: %v", err)
	}

	if err := UnmarshalMessage(m, &d); err != nil {
		t.Fatalf("Unexpected error unmarshaling: %v", err)
	}

	if !reflect.DeepEqual(want, d) {
		t.Fatalf("UnmarshalMessage did not preserve data:\nwant: %+v\ngot: %+v", want, d)
	}
}

func TestUnmarshalMessageNestedNilPtr(t *testing.T) {
	tm := &testMessage{
		Message: NewMessage(),
	}

	err := UnmarshalMessage(goldMarshaled, tm)
	if err != nil {
		t.Fatalf("Unexpected error unmarshaling: %v", err)
	}

	if !reflect.DeepEqual(*tm, goldUnmarshaled) {
		t.Fatalf("Unmarshaled message does not equal gold struct.\nExpected: %+v\nReceived: %+v", goldUnmarshaled, *tm)
	}
}

func TestMessageGet(t *testing.T) {
	v := goldMessage.Get("key1")
	if value, ok := v.(string); !ok {
		t.Fatalf("Expected %v to be string: received %T", value, value)
	} else if value != "value1" {
		t.Fatalf("Expected 'key1' to be 'value1': received %v", value)
	}

	v = goldMessage.Get("invalid")
	if v != nil {
		t.Fatalf("Expected nil for Get on non-existent key: received %v", v)
	}
}

func TestMessageSet(t *testing.T) {
	// Test that all supported types can be set.
	valid := []any{
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
		map[string]any{
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

func TestMessageUnset(t *testing.T) {
	m1 := &Message{
		keys: []string{"test1", "test2", "test3", "test4"},
		data: map[string]any{
			"test1": 1,
			"test2": 2,
			"test3": 3,
			"test4": 4,
		},
	}

	m2 := &Message{
		keys: []string{"test1", "test2", "test4"},
		data: map[string]any{
			"test1": 1,
			"test2": 2,
			"test4": 4,
		},
	}

	m1.Unset("test3")

	if !reflect.DeepEqual(m1, m2) {
		t.Fatalf("Unset corrupted message, expected: %+v\ngot: %+v\n", m2, m1)
	}

	m1.Unset("invalid")

	if !reflect.DeepEqual(m1, m2) {
		t.Fatalf("Unset with non-existent key was not a no-op, expected: %+v\ngot: %+v\n", m2, m1)
	}
}

func TestMessageSetTypeConversion(t *testing.T) {
	type conversion struct {
		in  any
		out any
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

func ExampleMessage_Set() {
	m := NewMessage()

	if err := m.Set("version", 2); err != nil {
		fmt.Println(err)
		return
	}

	if err := m.Set("mobike", false); err != nil {
		fmt.Println(err)
		return
	}

	if err := m.Set("local_addrs", []string{"192.168.0.1/24"}); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(m)
	// Output: {
	//   version = 2
	//   mobike = no
	//   local_addrs = 192.168.0.1/24
	// }
}

func TestEmptyMessageElement(t *testing.T) {
	elements := map[reflect.Value]bool{
		/* bool's should never be empty */
		reflect.ValueOf(false): false,
		reflect.ValueOf(true):  false,
		/* integer types should never be empty */
		reflect.ValueOf(int(0)):    false,
		reflect.ValueOf(int(1)):    false,
		reflect.ValueOf(int8(0)):   false,
		reflect.ValueOf(int8(1)):   false,
		reflect.ValueOf(int16(0)):  false,
		reflect.ValueOf(int16(1)):  false,
		reflect.ValueOf(int32(0)):  false,
		reflect.ValueOf(int32(1)):  false,
		reflect.ValueOf(int64(0)):  false,
		reflect.ValueOf(int64(1)):  false,
		reflect.ValueOf(uint(0)):   false,
		reflect.ValueOf(uint(1)):   false,
		reflect.ValueOf(uint8(0)):  false,
		reflect.ValueOf(uint8(1)):  false,
		reflect.ValueOf(uint16(0)): false,
		reflect.ValueOf(uint16(1)): false,
		reflect.ValueOf(uint32(0)): false,
		reflect.ValueOf(uint32(1)): false,
		reflect.ValueOf(uint64(0)): false,
		reflect.ValueOf(uint64(1)): false,
		/* empty strings should be empty, otherwise not */
		reflect.ValueOf(""):     true,
		reflect.ValueOf("test"): false,
		/* nil or zero-length slices should be empty */
		reflect.ValueOf([]string{"test1", "test2"}): false,
		reflect.ValueOf([]string{}):                 true,
		reflect.ValueOf([]string(nil)):              true,
		/* nil or zero-length maps should be empty */
		reflect.ValueOf(map[string]any{"test": 0}): false,
		reflect.ValueOf(map[string]any{}):          true,
		reflect.ValueOf(map[string]any(nil)):       true,
		/* a struct is empty if each of its fields are empty */
		reflect.ValueOf(struct{ Test string }{"test"}): false,
		reflect.ValueOf(struct{ Test string }{}):       true,
		/* nil pointers should be empty */
		reflect.ValueOf((*bool)(nil)):                             true,
		reflect.ValueOf(func() *bool { b := false; return &b }()): false,
	}

	for k, v := range elements {
		if emptyMessageElement(k) != v {
			s := "empty"
			if v {
				s = "non-empty"
			}

			t.Errorf("emptyMessageElement reports %v (%s) is %s", k.Interface(), k.Kind(), s)
		}
	}
}
