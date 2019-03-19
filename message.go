// Package vici implements a strongSwan vici protocol client
package vici

import (
	"bytes"
	"errors"
	"io"
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
	buf := bytes.NewBuffer([]byte{})

	for k, v := range m.data {
		rv := reflect.ValueOf(v)

		var (
			data []byte
			err  error
		)

		switch rv.Kind() {

		case reflect.String:
			uv := v.(string)

			data, err = m.encodeKeyValue(k, uv)
			if err != nil {
				return []byte{}, err
			}

		case reflect.Slice, reflect.Array:
			uv := v.([]string)

			data, err = m.encodeList(k, uv)
			if err != nil {
				return []byte{}, err
			}

		case reflect.Map:
			uv := v.(map[string]interface{})

			data, err = m.encodeSection(k, uv)
			if err != nil {
				return []byte{}, err
			}

		default:
			return []byte{}, errors.New("unsupported data type")
		}

		_, err = buf.Write(data)
		if err != nil {
			return []byte{}, err
		}
	}

	return buf.Bytes(), nil
}

func (m *message) decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	b, err := buf.ReadByte()
	if err != nil {
		return err
	}

	for buf.Len() > 0 {
		// Determine the next message element
		switch b {

		case msgKeyValue:
			n, err := m.decodeKeyValue(buf.Bytes())
			if err != nil {
				return err
			}
			buf.Next(n)

		case msgListStart:
			n, err := m.decodeList(buf.Bytes())
			if err != nil {
				return err
			}
			buf.Next(n)

		case msgSectionStart:
			n, err := m.decodeSection(buf.Bytes())
			if err != nil {
				return err
			}
			buf.Next(n)
		}

		b, err = buf.ReadByte()
		if err != nil && err != io.EOF {
			return err
		}
	}

	return nil
}

// encodeKeyValue will return a byte slice of an encoded key-value pair.
//
// The size of the byte slice is the length of the key and value, plus four bytes:
// one byte for message element type, one byte for key length, and two bytes for value
// length.
func (m *message) encodeKeyValue(key, value string) ([]byte, error) {
	// Initialize buffer to indictate the message element type
	// is a key-value pair
	buf := bytes.NewBuffer([]byte{msgKeyValue})

	// Write the key length and key
	err := buf.WriteByte(uint8(len(key)))
	if err != nil {
		return []byte{}, err
	}

	_, err = buf.WriteString(key)
	if err != nil {
		return []byte{}, err
	}

	// Write the value's length to the buffer as two bytes
	vl := []byte{
		uint8(len(value) >> 8),
		uint8(len(value) & 0xff),
	}
	_, err = buf.Write(vl)
	if err != nil {
		return []byte{}, err
	}

	// Write the value to the buffer
	_, err = buf.WriteString(value)
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// encodeList will return a byte slice of an encoded list.
//
// The size of the byte slice is the length of the key and total length of
// the list (sum of length of the items in the list), plus three bytes for each
// list item: one for message element type, and two for item length. Another three
// bytes are used to indicate list start and list stop, and the length of the key.
func (m *message) encodeList(key string, list []string) ([]byte, error) {
	// Initialize buffer to indictate the message element type
	// is the start of a list
	buf := bytes.NewBuffer([]byte{msgListStart})

	// Write the key length and key
	err := buf.WriteByte(uint8(len(key)))
	if err != nil {
		return []byte{}, err
	}

	_, err = buf.WriteString(key)
	if err != nil {
		return []byte{}, err
	}

	for _, item := range list {
		// Indicate that this is a list item
		err = buf.WriteByte(uint8(msgListItem))

		// Write the item's length to the buffer as two bytes
		il := []byte{
			uint8(len(item) >> 8),
			uint8(len(item) & 0xff),
		}
		_, err = buf.Write(il)
		if err != nil {
			return []byte{}, err
		}

		// Write the item to the buffer
		_, err = buf.WriteString(item)
		if err != nil {
			return []byte{}, err
		}
	}

	// Indicate the end of the list
	err = buf.WriteByte(uint8(msgListEnd))
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// encodeSection will return a byte slice of an encoded section
func (m *message) encodeSection(key string, section map[string]interface{}) ([]byte, error) {
	// Initialize buffer to indictate the message element type
	// is the start of a section
	buf := bytes.NewBuffer([]byte{msgSectionStart})

	// Write the key length and key
	err := buf.WriteByte(uint8(len(key)))
	if err != nil {
		return []byte{}, err
	}

	_, err = buf.WriteString(key)
	if err != nil {
		return []byte{}, err
	}

	// Encode the sections elements
	for k, v := range section {
		rv := reflect.ValueOf(v)

		var data []byte

		switch rv.Kind() {

		case reflect.String:
			uv := v.(string)

			data, err = m.encodeKeyValue(k, uv)
			if err != nil {
				return []byte{}, err
			}

		case reflect.Slice, reflect.Array:
			uv := v.([]string)

			data, err = m.encodeList(k, uv)
			if err != nil {
				return []byte{}, err
			}

		case reflect.Map:
			uv := v.(map[string]interface{})

			data, err = m.encodeSection(k, uv)
			if err != nil {
				return []byte{}, err
			}

		default:
			return []byte{}, errors.New("unsupported data type")
		}

		_, err = buf.Write(data)
		if err != nil {
			return []byte{}, err
		}

	}

	// Indicate the end of the section
	err = buf.WriteByte(uint8(msgSectionEnd))
	if err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// decodeKeyValue will decode a key-value pair and write it to the message's
// data, and returns the number of bytes decoded.
func (m *message) decodeKeyValue(data []byte) (int, error) {
	buf := bytes.NewBuffer(data)

	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return -1, err
	}

	keyLen := int(n)
	key := string(buf.Next(keyLen))
	if len(key) != keyLen {
		return -1, errors.New("expected key length does not match actual length")
	}

	// Read the value's length
	v := buf.Next(2)
	if len(v) != 2 {
		return -1, errors.New("unexpected end of buffer")

	}

	// Read the value from the buffer
	valueLen := int(v[0])<<8 + int(v[1])
	value := string(buf.Next(valueLen))
	if len(value) != valueLen {
		return -1, errors.New("expected value length does not match actual length")
	}

	m.data[key] = value

	// Return the length of the key and value, plus the three bytes for their
	// lengths
	return keyLen + valueLen + 3, nil
}

// decodeList will decode a list and write it to the message's data, and return
// the number of bytes decoded.
func (m *message) decodeList(data []byte) (int, error) {
	var list []string

	buf := bytes.NewBuffer(data)

	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return -1, err
	}

	keyLen := int(n)
	key := string(buf.Next(keyLen))
	if len(key) != keyLen {
		return -1, errors.New("expected key length does not match actual length")
	}

	b, err := buf.ReadByte()
	if err != nil {
		return -1, err
	}

	// Keep track of bytes decoded
	count := keyLen + 2

	// Read the list from the buffer
	for b != msgListEnd {
		// Ensure this is the beginning of a list item
		if b != msgListItem {
			return -1, errors.New("expected beginning of list item")
		}

		// Read the value's length
		v := buf.Next(2)
		if len(v) != 2 {
			return -1, errors.New("unexpected end of buffer")

		}

		// Read the value from the buffer
		valueLen := int(v[0])<<8 + int(v[1])
		value := string(buf.Next(valueLen))
		if len(value) != valueLen {
			return -1, errors.New("expected value length does not match actual length")
		}

		list = append(list, value)

		b, err = buf.ReadByte()
		if err != nil {
			return -1, err
		}

		count += valueLen + 3
	}

	m.data[key] = list

	return count, nil
}

// decodeSection will decode a section into a message's data, and return the number
// of bytes decoded.
func (m *message) decodeSection(data []byte) (int, error) {
	section := newMessage()

	buf := bytes.NewBuffer(data)

	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return -1, err
	}

	keyLen := int(n)
	key := string(buf.Next(keyLen))
	if len(key) != keyLen {
		return -1, errors.New("expected key length does not match actual length")
	}

	b, err := buf.ReadByte()
	if err != nil {
		return -1, err
	}

	// Keep track of bytes decoded
	count := keyLen + 2

	for b != msgSectionEnd {
		// Determine the next message element
		switch b {

		case msgKeyValue:
			n, err := section.decodeKeyValue(buf.Bytes())
			if err != nil {
				return -1, err
			}
			// Skip those decoded bytes
			buf.Next(n)

			count += n

		case msgListStart:
			n, err := section.decodeList(buf.Bytes())
			if err != nil {
				return -1, err
			}
			// Skip those decoded bytes
			buf.Next(n)

			count += n

		case msgSectionStart:
			n, err := section.decodeSection(buf.Bytes())
			if err != nil {
				return -1, err
			}
			// Skip those decoded bytes
			buf.Next(n)

			count += n

		default:
			return -1, errors.New("expected key-value pair or the beginning of a section or list")
		}

		b, err = buf.ReadByte()
		if err != nil {
			return -1, err
		}

		count += 1
	}

	m.data[key] = section.data

	return count, nil
}
