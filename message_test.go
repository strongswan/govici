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
