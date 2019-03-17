// Package vici implements a strongSwan vici protocol client
package vici

import (
	"reflect"
)

const (
	// Begin a new section having a name
	msgSectionStart = iota + 1

	// End a previously started section
	msgSectionEnd

	// Define a value for a named key in the current section
	msgKeyValue

	// Begin a name list for list items
	msgListStart

	// Dfeine an unnamed item value in the current list
	msgListItem

	// End a prevsiously started list
	msgListEnd
)

type message struct {
	data map[string]interface{}
}

func (m *message) encode() ([]byte, error) {
	b := make([]byte, 0)

	for k, v := range m.data {
		rv := reflect.ValueOf(v)

		switch rv.Kind() {

		case reflect.String:
			uv := v.(string)
			b = append(b, encodeKeyValue(k, uv)...)

		case reflect.Slice, reflect.Array:
			uv := v.([]string)
			b = append(b, encodeList(k, uv)...)

		case reflect.Map:
			uv := v.(map[string]interface{})
			b = append(b, encodeSection(k, uv)...)
		}
	}

	return b, nil
}

func (m *message) decode(data []byte) error {
	return nil
}

// encodeKeyValue will return a byte slice of an encoded key-value pair.
//
// The size of the byte slice is the length of the key and value, plus four bytes:
// one byte for message element type, one byte for key length, and two bytes for value
// length.
func encodeKeyValue(key, value string) []byte {
	keyLen := len(key)
	valueLen := len(value)

	b := make([]byte, keyLen+valueLen+4)

	// Indicate that the message element type is key-value
	b[0] = msgKeyValue

	// Add the key
	b[1] = uint8(keyLen)
	for i, v := range []byte(key) {
		b[i+2] = v
	}

	index := keyLen

	// Add the value
	b[index+2] = uint8(valueLen >> 8)
	b[index+3] = uint8(valueLen & 0xff)
	for i, v := range []byte(value) {
		b[index+i+4] = v
	}

	return b
}

// encodeList will return a byte slice of an encoded list.
//
// The size of the byte slice is the length of the key and total length of
// the list (sum of length of the items in the list), plus three bytes for each
// list item: one for message element type, and two for item length. Another three
// bytes are used to indicate list start and list stop, and the length of the key.
func encodeList(key string, list []string) []byte {
	listLen := len(key) + 3
	for _, v := range list {
		listLen += len(v) + 3
	}

	b := make([]byte, listLen)

	// Indicate that this is the start of a list
	b[0] = msgListStart

	// Add the list key
	b[1] = uint8(len(key))
	for i, v := range []byte(key) {
		b[i+2] = v
	}

	index := len(key) + 2
	for _, item := range list {
		itemLen := len(item)

		// Indicate a new list item
		b[index] = msgListItem
		b[index+1] = uint8(itemLen >> 8)
		b[index+2] = uint8(itemLen & 0xff)

		for i, v := range []byte(item) {
			b[index+i+3] = v
		}
		index += itemLen + 3
	}

	// Indicate the end of the list
	b[index] = msgListEnd

	return b
}

// encodeSection will return a byte slice of an encoded section
func encodeSection(key string, section map[string]interface{}) []byte {
	// Start with a byte slice big enough for section start and key. Append for
	// section elements.
	b := make([]byte, len(key)+2)

	// Indicate the start of a section
	b[0] = msgSectionStart

	// Add the section key
	b[1] = uint8(len(key))
	for i, v := range []byte(key) {
		b[i+2] = v
	}

	// Encode the sections elements
	for k, v := range section {
		rv := reflect.ValueOf(v)

		switch rv.Kind() {

		case reflect.String:
			uv := v.(string)
			b = append(b, encodeKeyValue(k, uv)...)

		case reflect.Slice, reflect.Array:
			uv := v.([]string)
			b = append(b, encodeList(k, uv)...)

		case reflect.Map:
			uv := v.(map[string]interface{})
			b = append(b, encodeSection(k, uv)...)

			// TODO: panic or return error on default?
		}
	}

	// Indicate the end of the section
	b = append(b, msgSectionEnd)

	return b
}
