// Package vici implements a strongSwan vici protocol client
package vici

import (
	"errors"
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

func newMessage() *message {
	return &message{
		data: make(map[string]interface{}),
	}
}

func (m *message) encode() ([]byte, error) {
	b := make([]byte, 0)

	for k, v := range m.data {
		rv := reflect.ValueOf(v)

		switch rv.Kind() {

		case reflect.String:
			uv := v.(string)
			b = append(b, m.encodeKeyValue(k, uv)...)

		case reflect.Slice, reflect.Array:
			uv := v.([]string)
			b = append(b, m.encodeList(k, uv)...)

		case reflect.Map:
			uv := v.(map[string]interface{})
			b = append(b, m.encodeSection(k, uv)...)
		}
	}

	return b, nil
}

func (m *message) decode(data []byte) error {
	index := 0

	for index < len(data) {
		// Determine the next message element
		switch data[index] {

		case msgKeyValue:
			index += m.decodeKeyValue(data[index+1:])

		case msgListStart:
			n, err := m.decodeList(data[index+1:])
			if err != nil {
				return err
			}
			index += n + 1

			if index >= len(data) {
				return errors.New("number of bytes decoded exceeds data length")
			}

			// Ensure that the list was completely decoded
			if data[index] != msgListEnd {
				return errors.New("expected end of list item")
			}

		case msgSectionStart:
			n, err := m.decodeSection(data[index+1:])
			if err != nil {
				return err
			}
			index += n + 1

			if index >= len(data) {
				return errors.New("number of bytes decoded exceeds data length")
			}

			// Ensure that the section was completely decoded
			if data[index] != msgSectionEnd {
				return errors.New("expected end of section")
			}
		}
		index += 1
	}

	// Ensure that all the data was decoded
	if index != len(data) {
		return errors.New("unexpected additional data after decode")
	}

	return nil
}

// encodeKeyValue will return a byte slice of an encoded key-value pair.
//
// The size of the byte slice is the length of the key and value, plus four bytes:
// one byte for message element type, one byte for key length, and two bytes for value
// length.
func (m *message) encodeKeyValue(key, value string) []byte {
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
func (m *message) encodeList(key string, list []string) []byte {
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

		// Provide the length of the item with 16 bytes
		b[index+1] = uint8(itemLen >> 8)
		b[index+2] = uint8(itemLen & 0xff)

		index += 3
		for i, v := range []byte(item) {
			b[index+i] = v
		}
		index += itemLen
	}

	// Indicate the end of the list
	b[index] = msgListEnd

	return b
}

// encodeSection will return a byte slice of an encoded section
func (m *message) encodeSection(key string, section map[string]interface{}) []byte {
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
			b = append(b, m.encodeKeyValue(k, uv)...)

		case reflect.Slice, reflect.Array:
			uv := v.([]string)
			b = append(b, m.encodeList(k, uv)...)

		case reflect.Map:
			uv := v.(map[string]interface{})
			b = append(b, m.encodeSection(k, uv)...)

			// TODO: panic or return error on default?
		}
	}

	// Indicate the end of the section
	b = append(b, msgSectionEnd)

	return b
}

// decodeKeyValue will decode a key-value pair and write it to the message's
// data, and returns the number of bytes decoded.
func (m *message) decodeKeyValue(data []byte) int {
	index := 1

	// Read the key
	keyLen := int(data[0])
	key := string(data[index : index+keyLen])
	index += keyLen

	// Read the value
	valueLen := int(data[index])<<8 + int(data[index+1])
	index += 2

	value := string(data[index : index+valueLen])

	// Write the key-value pair
	m.data[key] = value

	// Return the length of the key and value, plus the three bytes for their
	// lengths
	return keyLen + valueLen + 3
}

// decodeList will decode a list and write it to the message's data, and return
// the number of bytes decoded.
func (m *message) decodeList(data []byte) (int, error) {
	var list []string

	index := 1

	// Read the key
	keyLen := int(data[0])
	key := string(data[index : index+keyLen])
	index += keyLen

	for data[index] != msgListEnd {
		// Ensure this is the beginning of a list item
		if data[index] != msgListItem {
			return -1, errors.New("expected beginning of list item")
		}
		itemLen := int(data[index+1])<<8 + int(data[index+2])
		index += 3

		list = append(list, string(data[index:index+itemLen]))

		// Set the new index
		index += itemLen
	}

	m.data[key] = list

	return index, nil
}

// decodeSection will decode a section into a message's data, and return the number
// of bytes decoded.
func (m *message) decodeSection(data []byte) (int, error) {
	section := newMessage()

	index := 1

	// Read the key
	keyLen := int(data[0])
	key := string(data[index : index+keyLen])
	index += keyLen

	for data[index] != msgSectionEnd {
		// Determine the next message element
		switch data[index] {

		case msgKeyValue:
			n := section.decodeKeyValue(data[index+1:])
			index += n + 1

		case msgListStart:
			n, err := section.decodeList(data[index+1:])
			if err != nil {
				return -1, err
			}
			index += n + 1

			// Ensure that the list was completely decoded
			if data[index] != msgListEnd {
				return -1, errors.New("expected end of list item")
			}
			index += 1

		case msgSectionStart:
			n, err := section.decodeSection(data[index+1:])
			if err != nil {
				return -1, err
			}
			index += n + 1

			// Ensure that the section was completely decoded
			if data[index] != msgSectionEnd {
				return -1, errors.New("expected end of section")
			}
			index += 1

		default:
			return -1, errors.New("expected key-value pair or the beginning of a section or list")
		}
	}

	m.data[key] = section.data

	return index, nil
}
