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
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

const (
	// Begin a new section having a name
	msgSectionStart uint8 = iota + 1

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

var (
	// Generic encoding/decoding and marshaling/unmarshaling errors
	errEncoding  = errors.New("vici: error encoding message")
	errDecoding  = errors.New("vici: error decoding message")
	errMarshal   = errors.New("vici: error marshaling message")
	errUnmarshal = errors.New("vici: error unmarshaling message")

	// Encountered unsupported type when encoding a message
	errUnsupportedType = errors.New("vici: unsupported message element type")

	// Used in CheckError - the 'success' field was set to "no"
	errCommandFailed = errors.New("vici: command failed")

	// Base message for decoding errors that are due to an incorrectly formatted message
	errMalformedMessage = errors.New("vici: malformed message")

	// Malformed message errors
	errBadKey            = fmt.Errorf("%v: expected key length does not match actual length", errMalformedMessage)
	errBadValue          = fmt.Errorf("%v: expected value length does not match actual length", errMalformedMessage)
	errEndOfBuffer       = fmt.Errorf("%v: unexpected end of buffer", errMalformedMessage)
	errExpectedBeginning = fmt.Errorf("%v: expected beginning of message element", errMalformedMessage)

	// Marshaling errors
	errMarshalUnsupportedType = fmt.Errorf("%v: encountered unsupported type", errMarshal)

	// Unmarshaling errors
	errUnmarshalBadType         = fmt.Errorf("%v: type must be non-nil pointer or map", errUnmarshal)
	errUnmarshalTypeMismatch    = fmt.Errorf("%v: incompatible types", errUnmarshal)
	errUnmarshalNonMessage      = fmt.Errorf("%v: encountered non-message type", errUnmarshal)
	errUnmarshalUnsupportedType = fmt.Errorf("%v: encountered unsupported type", errUnmarshal)
	errUnmarshalParseFailure    = fmt.Errorf("%v: failed to parse value", errUnmarshal)
)

// MessageStream is used to feed continuous data during a command request, and simply
// contains a slice of *Message.
type MessageStream struct {
	// Message list
	messages []*Message
}

// Messages returns the messages received from the streamed request.
func (ms *MessageStream) Messages() []*Message {
	return ms.messages
}

// Message represents a vici message as described in the vici README:
//
//     https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html
//
// A message supports encoding key-value pairs, lists, and sub-sections (or sub-messages).
// Within a Message, each value, list, and sub-section is keyed by a string.
//
// The value in a key-value pair is represented by a string, lists are represented by a string slice,
// and sub-sections are represented by *Message. When constructing a Message, other types may be used
// for convenience, and may have rules on how they are converted to an appropriate internal message
// element type. See Message.Set and MarshalMessage for details.
type Message struct {
	keys []string

	data map[string]interface{}
}

// NewMessage returns an empty Message.
func NewMessage() *Message {
	return &Message{
		keys: make([]string, 0),
		data: make(map[string]interface{}),
	}
}

// MarshalMessage returns a Message encoded from v. The type of v must be either a map,
// struct, or struct pointer.
//
// If v is a map, the map's key type must be a string, and the type of the corresponding map element
// must be supported by Message.Set or MarshalMessage itself.
//
// If v is a struct or points to one, fields are only marshaled if they are exported and explicitly
// have a vici struct tag set. In these cases, the struct tag defines the key used for that field in
// the Message, and the field type must be supported by Message.Set or MarshalMessage itself.
// Embedded structs may be used as fields, either by explicitly giving them a field name with the vici
// struct tag, or by marking them as inline. Inlined structs are defined by using the opt "inline"
// on the vici tag, for example: `vici:",inline"`.
//
// Any field that would be encoded as an empty message element is always omitted during marshaling. However,
// "empty message element" is not necessarily analogous to a type's zero value in Go. Because all supported
// Go types are marshaled to either string, []string, or *Message, a field is considered an empty message
// element if it would be encoded as either an empty string, zero-length []string or nil. On the other hand,
// an integer's zero value is 0, but this is marshaled to the string "0", and will not be omitted from marshaling.
// Likewise, a bool's zero value is false, which is marshaled to the string "no".
func MarshalMessage(v interface{}) (*Message, error) {
	m := NewMessage()
	if err := m.marshal(v); err != nil {
		return nil, err
	}

	return m, nil
}

// UnmarshalMessage unmarshals m to a map or struct (or struct pointer).
// When unmarshaling to a struct, only exported fields with a vici struct tag
// explicitly set are unmarshaled. Struct fields can be unmarshaled inline
// by providing the opt "inline" to the vici struct tag.
//
// An error is returned if the underlying value of v cannot be unmarshaled into, or
// an unsupported type is encountered.
func UnmarshalMessage(m *Message, v interface{}) error {
	return m.unmarshal(v)
}

// Set sets key to value. An error is returned if the underlying type of
// v is not supported.
//
// If the type of v is supported, it is represented in the message as either a
// string, []string, or *Message. The currently supported types are:
//
//  - string
//  - integer types (converted to string)
//  - bool (where true and false are converted to the strings "yes" and "no", respectively)
//  - []string
//  - *Message
//  - map (the map must be valid as per MarshalMessage)
//  - struct (the struct must be valid as per MarshalMessage)
//
// Pointer types of the above are allowed and can be used to differentiate between
// an unset value and a zero value. If a pointer is nil, it is not added to the message.
//
// If the key already exists the value is overwritten, but the ordering
// of the message is not changed.
func (m *Message) Set(key string, value interface{}) error {
	return m.marshalField(key, reflect.ValueOf(value))
}

// Unset unsets the message field identified by key. There is no effect if the
// key does not exist.
func (m *Message) Unset(key string) {
	for i, v := range m.keys {
		if v != key {
			continue
		}

		// If we found the key, delete it from the message
		// keys while preserving order, and delete the value
		// from the map.
		m.keys = append(m.keys[:i], m.keys[i+1:]...)
		delete(m.data, key)

		return
	}
}

// Get returns the value of the field identified by the last key in keys, if it
// exists. If the field does not exist, nil is returned. It is expected that
// each intermediate key will return a sub-section (*Message). Else, nil is
// returned.
//
// The value returned by Get is the internal message representation of that
// field, which means the type is either string, []string, or *Message.
func (m *Message) Get(keys ...string) interface{} {
	tmp := new(Message)
	*tmp = *m

	for i, k := range keys {
		v, ok := tmp.data[k]
		if !ok {
			return nil
		}

		// If this is the last key, return whatever is in data[k].
		// Otherwise, reset tmp and continue the loop.
		if i == len(keys)-1 {
			return v
		}

		tmp, ok = v.(*Message)
		if !ok {
			return nil
		}
	}

	return nil
}

// Keys returns the list of valid message keys.
func (m *Message) Keys() []string {
	keys := make([]string, len(m.keys))
	copy(keys, m.keys)

	return keys
}

// Err examines a command response Message, and determines if it was successful.
// If it was, or if the message does not contain a 'success' field, nil is returned. Otherwise,
// an error is returned using the 'errmsg' field.
func (m *Message) Err() error {
	if success, ok := m.data["success"]; ok {
		if success != "yes" {
			return fmt.Errorf("%v: %v", errCommandFailed, m.data["errmsg"])
		}
	}

	return nil
}

func (m *Message) addItem(key string, value interface{}) error {
	// Check if the key is already set in the message
	_, exists := m.data[key]

	switch v := value.(type) {
	case string:
		m.data[key] = v
	case []string:
		m.data[key] = v
	case *Message:
		m.data[key] = v
	default:
		return errUnsupportedType
	}

	// Only append to keys if this is a new key.
	if !exists {
		m.keys = append(m.keys, key)
	}

	return nil
}

type messageElement struct {
	k string
	v interface{}
}

func (m *Message) elements() []messageElement {
	ordered := make([]messageElement, len(m.keys))

	for i, k := range m.keys {
		ordered[i] = messageElement{k: k, v: m.data[k]}
	}

	return ordered
}

func (m *Message) encode() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	for _, e := range m.elements() {
		k := e.k
		v := e.v

		rv := reflect.ValueOf(v)

		var (
			data []byte
			err  error
		)

		switch rv.Kind() {
		// In these cases, the variable 'uv' is short for
		// 'underlying value.'
		case reflect.String:
			uv := v.(string)

			data, err = m.encodeKeyValue(k, uv)
			if err != nil {
				return nil, err
			}

		case reflect.Slice, reflect.Array:
			uv := v.([]string)

			data, err = m.encodeList(k, uv)
			if err != nil {
				return nil, err
			}

		case reflect.Ptr:
			uv, ok := v.(*Message)
			if !ok {
				return nil, errUnsupportedType
			}

			data, err = m.encodeSection(k, uv)
			if err != nil {
				return nil, err
			}

		default:
			return nil, errUnsupportedType
		}

		_, err = buf.Write(data)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errEncoding, err)
		}
	}

	return buf.Bytes(), nil
}

func (m *Message) decode(data []byte) error {
	buf := bytes.NewBuffer(data)

	b, err := buf.ReadByte()
	if err != nil && err != io.EOF {
		return fmt.Errorf("%v: %v", errDecoding, err)
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
			return fmt.Errorf("%v: %v", errDecoding, err)
		}
	}

	return nil
}

// encodeKeyValue will return a byte slice of an encoded key-value pair.
//
// The size of the byte slice is the length of the key and value, plus four bytes:
// one byte for message element type, one byte for key length, and two bytes for value
// length.
func (m *Message) encodeKeyValue(key, value string) ([]byte, error) {
	// Initialize buffer to indictate the message element type
	// is a key-value pair
	buf := bytes.NewBuffer([]byte{msgKeyValue})

	// Write the key length and key
	err := buf.WriteByte(uint8(len(key)))
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	_, err = buf.WriteString(key)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	// Write the value's length to the buffer as two bytes
	vl := make([]byte, 2)
	binary.BigEndian.PutUint16(vl, uint16(len(value)))

	_, err = buf.Write(vl)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	// Write the value to the buffer
	_, err = buf.WriteString(value)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	return buf.Bytes(), nil
}

// encodeList will return a byte slice of an encoded list.
//
// The size of the byte slice is the length of the key and total length of
// the list (sum of length of the items in the list), plus three bytes for each
// list item: one for message element type, and two for item length. Another three
// bytes are used to indicate list start and list stop, and the length of the key.
func (m *Message) encodeList(key string, list []string) ([]byte, error) {
	// Initialize buffer to indictate the message element type
	// is the start of a list
	buf := bytes.NewBuffer([]byte{msgListStart})

	// Write the key length and key
	err := buf.WriteByte(uint8(len(key)))
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	_, err = buf.WriteString(key)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	for _, item := range list {
		// Indicate that this is a list item
		err = buf.WriteByte(msgListItem)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errEncoding, err)
		}

		// Write the item's length to the buffer as two bytes
		il := make([]byte, 2)
		binary.BigEndian.PutUint16(il, uint16(len(item)))

		_, err = buf.Write(il)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errEncoding, err)
		}

		// Write the item to the buffer
		_, err = buf.WriteString(item)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errEncoding, err)
		}
	}

	// Indicate the end of the list
	err = buf.WriteByte(msgListEnd)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	return buf.Bytes(), nil
}

// encodeSection will return a byte slice of an encoded section
func (m *Message) encodeSection(key string, section *Message) ([]byte, error) {
	// Initialize buffer to indictate the message element type
	// is the start of a section
	buf := bytes.NewBuffer([]byte{msgSectionStart})

	// Write the key length and key
	err := buf.WriteByte(uint8(len(key)))
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	_, err = buf.WriteString(key)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	// Encode the sections elements
	for _, e := range section.elements() {
		k := e.k
		v := e.v

		rv := reflect.ValueOf(v)

		var data []byte

		switch rv.Kind() {
		case reflect.String:
			uv := v.(string)

			data, err = m.encodeKeyValue(k, uv)
			if err != nil {
				return nil, err
			}

		case reflect.Slice, reflect.Array:
			uv := v.([]string)

			data, err = m.encodeList(k, uv)
			if err != nil {
				return nil, err
			}

		case reflect.Ptr:
			uv, ok := v.(*Message)
			if !ok {
				return nil, errUnsupportedType
			}

			data, err = m.encodeSection(k, uv)
			if err != nil {
				return nil, err
			}

		default:
			return nil, errUnsupportedType
		}

		_, err = buf.Write(data)
		if err != nil {
			return nil, fmt.Errorf("%v: %v", errEncoding, err)
		}
	}

	// Indicate the end of the section
	err = buf.WriteByte(msgSectionEnd)
	if err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	return buf.Bytes(), nil
}

// decodeKeyValue will decode a key-value pair and write it to the message's
// data, and returns the number of bytes decoded.
func (m *Message) decodeKeyValue(data []byte) (int, error) {
	buf := bytes.NewBuffer(data)

	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
	}

	keyLen := int(n)
	key := string(buf.Next(keyLen))
	if len(key) != keyLen {
		return -1, errBadKey
	}

	// Read the value's length
	v := buf.Next(2)
	if len(v) != 2 {
		return -1, errEndOfBuffer
	}

	// Read the value from the buffer
	valueLen := int(binary.BigEndian.Uint16(v))
	value := string(buf.Next(valueLen))
	if len(value) != valueLen {
		return -1, errBadValue
	}

	err = m.addItem(key, value)
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
	}

	// Return the total number of bytes read. Specifically,
	// we have 1 byte for the key length, n (keyLen) bytes
	// for the key itself, 2 bytes for the value length, and
	// k (valueLen) bytes for the value itself.
	return keyLen + valueLen + 3, nil
}

// decodeList will decode a list and write it to the message's data, and return
// the number of bytes decoded.
func (m *Message) decodeList(data []byte) (int, error) {
	var list []string

	buf := bytes.NewBuffer(data)

	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
	}

	keyLen := int(n)
	key := string(buf.Next(keyLen))
	if len(key) != keyLen {
		return -1, errBadKey
	}

	b, err := buf.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
	}

	// Start a counter to keep track of bytes decoded.
	//
	// So far, we've read one byte for the key length,
	// n (keyLen) bytes for the key itself, and one byte
	// to start the first list item.
	count := keyLen + 2

	// Read the list from the buffer
	for b != msgListEnd {
		// Ensure this is the beginning of a list item
		if b != msgListItem {
			return -1, errExpectedBeginning
		}

		// Read the value's length
		v := buf.Next(2)
		if len(v) != 2 {
			return -1, errEndOfBuffer
		}

		// Read the value from the buffer
		valueLen := int(binary.BigEndian.Uint16(v))
		value := string(buf.Next(valueLen))
		if len(value) != valueLen {
			return -1, errBadValue
		}

		list = append(list, value)

		b, err = buf.ReadByte()
		if err != nil {
			return -1, fmt.Errorf("%v: %v", errDecoding, err)
		}

		// In this iteration, we've read 2 bytes to get the
		// length of the list item, n (valueLen) bytes for
		// the value itself, and one more byte to either
		// (a) start the next list item, or (b) end the list.
		count += valueLen + 3
	}

	err = m.addItem(key, list)
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
	}

	return count, nil
}

// decodeSection will decode a section into a message's data, and return the number
// of bytes decoded.
func (m *Message) decodeSection(data []byte) (int, error) {
	section := NewMessage()

	buf := bytes.NewBuffer(data)

	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
	}

	keyLen := int(n)
	key := string(buf.Next(keyLen))
	if len(key) != keyLen {
		return -1, errBadKey
	}

	b, err := buf.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("%v: %v", errDecoding, err)
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
			return -1, errExpectedBeginning
		}

		b, err = buf.ReadByte()
		if err != nil {
			return -1, fmt.Errorf("%v: %v", errDecoding, err)
		}

		count++
	}

	err = m.addItem(key, section)
	if err != nil {
		return -1, err
	}

	return count, nil
}

// messageTag is used for parsing struct tags in marshaling Messages
type messageTag struct {
	name   string
	skip   bool
	inline bool
}

func newMessageTag(tag reflect.StructTag) messageTag {
	t := tag.Get("vici")

	opts := strings.Split(t, ",")
	if len(opts) == 0 {
		return messageTag{skip: true}
	}

	mt := messageTag{name: opts[0]}
	for _, opt := range opts[1:] {
		if opt == "inline" {
			mt.inline = true
		}
	}

	if (!mt.inline && mt.name == "") || mt.name == "-" {
		mt.skip = true
	}

	return mt
}

func emptyMessageElement(rv reflect.Value) bool {
	switch rv.Kind() {
	case reflect.Slice:
		return rv.IsNil() || rv.Len() == 0

	case reflect.Struct:
		z := true
		for i := 0; i < rv.NumField(); i++ {
			z = z && emptyMessageElement(rv.Field(i))
		}
		return z

	case reflect.Ptr, reflect.String:
		return rv.Interface() == reflect.Zero(rv.Type()).Interface()

	case reflect.Map:
		return rv.IsNil() || len(rv.MapKeys()) == 0

	// The rest of the types checked here are ALWAYS considered non-empty, because
	// they will be encoded in their appropriate string representations, which are
	// always non-empty.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return false

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return false

	case reflect.Bool:
		return false
	}

	return false
}

func (m *Message) marshal(v interface{}) error {
	rv := reflect.ValueOf(v)

	if rv.Kind() == reflect.Ptr {
		rv = reflect.Indirect(rv)
	}

	switch rv.Kind() {
	case reflect.Struct:
		return m.marshalFromStruct(rv)

	case reflect.Map:
		return m.marshalFromMap(rv)

	default:
		return fmt.Errorf("%v: %v", errMarshalUnsupportedType, rv.Kind())
	}
}

func (m *Message) marshalFromStruct(rv reflect.Value) error {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		rf := rt.Field(i)

		mt := newMessageTag(rf.Tag)
		if mt.skip {
			continue
		}

		rfv := rv.Field(i)
		if !rfv.CanInterface() {
			continue
		}

		if mt.inline {
			if rfv.Kind() != reflect.Struct {
				return fmt.Errorf("%v: cannot marshal non-struct inlined field %v", errMarshalUnsupportedType, rv.Kind())
			}

			err := m.marshalFromStruct(rfv)
			if err != nil {
				return err
			}
			continue
		}

		if emptyMessageElement(rfv) {
			continue
		}

		// Add the message element
		err := m.marshalField(mt.name, rfv)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Message) marshalFromMap(rv reflect.Value) error {
	keys := rv.MapKeys()
	for _, k := range keys {
		v := rv.MapIndex(k)

		err := m.marshalField(k.String(), v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Message) marshalField(name string, rv reflect.Value) error {
	if rv.Kind() == reflect.Interface {
		rv = reflect.ValueOf(rv.Interface())
	}

	switch rv.Kind() {
	case reflect.String:
		return m.addItem(name, rv.String())

	case reflect.Slice, reflect.Array:
		return m.addItem(name, rv.Interface())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return m.addItem(name, strconv.FormatInt(rv.Int(), 10))

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return m.addItem(name, strconv.FormatUint(rv.Uint(), 10))

	case reflect.Bool:
		if rv.Bool() {
			return m.addItem(name, "yes")
		}
		return m.addItem(name, "no")

	case reflect.Ptr:
		if _, ok := rv.Interface().(*Message); ok {
			return m.addItem(name, rv.Interface())
		}
		return m.marshalField(name, reflect.Indirect(rv))

	case reflect.Struct, reflect.Map:
		msg := NewMessage()
		if err := msg.marshal(rv.Interface()); err != nil {
			return err
		}

		return m.addItem(name, msg)

	default:
		return fmt.Errorf("%v: %v", errMarshalUnsupportedType, rv.Kind())
	}
}

func (m *Message) unmarshal(v interface{}) error {
	rv := reflect.ValueOf(v)

	switch rv.Kind() {
	case reflect.Map:
		// Must be a non-nil map.
		if rv.IsNil() {
			return errUnmarshalBadType
		}

		return m.unmarshalToMap(rv)

	case reflect.Ptr:
		// Must be a pointer to a struct.
		if rv.IsNil() {
			return errUnmarshalBadType
		}

		rv = reflect.Indirect(rv)
		if rv.Kind() != reflect.Struct {
			return fmt.Errorf("%v: cannot unmarshal into non-struct pointer %v", errUnmarshalUnsupportedType, rv.Kind())
		}

		return m.unmarshalToStruct(rv)

	default:
		return fmt.Errorf("%v: cannot unmarshal into %v", errUnmarshalUnsupportedType, rv.Kind())
	}
}

func (m *Message) unmarshalToStruct(rv reflect.Value) error {
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		rf := rt.Field(i)
		tag := newMessageTag(rf.Tag)

		rfv := rv.Field(i)
		if !rfv.CanInterface() {
			continue
		}

		if tag.inline {
			if rfv.Kind() != reflect.Struct {
				return fmt.Errorf("%v: cannot unmarshal into non-struct inlined field %v", errUnmarshalUnsupportedType, rv.Kind())
			}

			err := m.unmarshalToStruct(rfv)
			if err != nil {
				return err
			}
			continue
		}

		value, ok := m.data[tag.name]
		if !ok {
			continue
		}

		err := m.unmarshalField(rfv, reflect.ValueOf(value))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Message) unmarshalToMap(rv reflect.Value) error {
	if rv.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("%v: map keys of type %v are not compatible with string", errUnmarshalTypeMismatch, rv.Type().Key().Kind())
	}

	for _, e := range m.elements() {
		key := reflect.ValueOf(e.k)
		val := reflect.ValueOf(e.v)

		rfv := rv.MapIndex(key)
		mapElemType := rv.Type().Elem()

		// We can't set the value of a map value directly since they are not
		// addressable. Thus we must create a new value element and set that
		// to the map instead.
		switch mapElemType.Kind() {
		case reflect.Ptr:
			if !rfv.IsValid() {
				rfv = reflect.New(mapElemType.Elem())
			}

		case reflect.Struct:
			if !rfv.IsValid() {
				rfv = reflect.Indirect(reflect.New(mapElemType))
			}

		case reflect.Slice:
			if !rfv.IsValid() {
				rfv = reflect.MakeSlice(mapElemType, 0, 0)
			}

		case reflect.Map:
			if !rfv.IsValid() {
				rfv = reflect.MakeMap(mapElemType)
			}

		case reflect.Interface:
			return fmt.Errorf("%v: map values cannot be generic interfaces", errUnmarshalTypeMismatch)

		default:
			rfv = reflect.Indirect(reflect.New(mapElemType))
		}

		err := m.unmarshalField(rfv, val)
		if err != nil {
			return err
		}

		rv.SetMapIndex(key, rfv)
	}

	return nil
}

func (m *Message) unmarshalField(field reflect.Value, rv reflect.Value) error {
	switch field.Kind() {
	case reflect.String:
		if _, ok := rv.Interface().(string); !ok {
			return fmt.Errorf("%v: string and %v", errUnmarshalTypeMismatch, rv.Type())
		}
		field.SetString(rv.String())

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		raw, ok := rv.Interface().(string)
		if !ok {
			return fmt.Errorf("%v: string and %v", errUnmarshalTypeMismatch, rv.Type())
		}

		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("%v: %v as %v", errUnmarshalParseFailure, raw, field.Type())
		}

		field.SetInt(parsed)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		raw, ok := rv.Interface().(string)
		if !ok {
			return fmt.Errorf("%v: string and %v", errUnmarshalTypeMismatch, rv.Type())
		}

		parsed, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("%v: %v as %v", errUnmarshalParseFailure, raw, field.Type())
		}

		field.SetUint(parsed)

	case reflect.Bool:
		raw, ok := rv.Interface().(string)
		if !ok {
			return fmt.Errorf("%v: string and %v", errUnmarshalTypeMismatch, rv.Type())
		}

		switch strings.ToLower(raw) {
		case "yes":
			field.SetBool(true)

		case "no":
			field.SetBool(false)

		default:
			return fmt.Errorf("%v: %v as %v", errUnmarshalParseFailure, raw, field.Type())
		}

	case reflect.Slice:
		if _, ok := rv.Interface().([]string); !ok {
			return fmt.Errorf("%v: []string and %v", errUnmarshalTypeMismatch, rv.Type())
		}
		field.Set(rv)

	case reflect.Ptr:
		if _, ok := field.Interface().(*Message); ok {
			field.Set(rv)

			return nil
		}

		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}

		return m.unmarshalField(field.Elem(), rv)

	case reflect.Struct:
		msg, ok := rv.Interface().(*Message)
		if !ok {
			return fmt.Errorf("%v: %v", errUnmarshalNonMessage, rv.Type())
		}

		fp := reflect.New(field.Type())
		if err := msg.unmarshal(fp.Interface()); err != nil {
			return err
		}

		field.Set(reflect.Indirect(fp))

	case reflect.Map:
		msg, ok := rv.Interface().(*Message)
		if !ok {
			return fmt.Errorf("%v: %v", errUnmarshalNonMessage, rv.Type())
		}

		fp := reflect.MakeMap(field.Type())
		if err := msg.unmarshal(fp.Interface()); err != nil {
			return err
		}

		field.Set(fp)

	default:
		return fmt.Errorf("%v: %v", errUnmarshalUnsupportedType, field.Kind())
	}

	return nil
}
