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
	"iter"
	"net"
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

const (
	// A name request message
	pktCmdRequest uint8 = iota

	// An unnamed response message for a request
	pktCmdResponse

	// An unnamed response if requested command is unknown
	pktCmdUnknown

	// A named event registration request
	pktEventRegister

	// A name event deregistration request
	pktEventUnregister

	// An unnamed response for successful event (de-)registration
	pktEventConfirm

	// An unnamed response if event (de-)registration failed
	pktEventUnknown

	// A named event message
	pktEvent

	// Used to indicate boundary of valid packet types
	pktInvalid
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

// Message represents a vici message as described in the vici README:
//
//	https://github.com/strongswan/strongswan/blob/master/src/libcharon/plugins/vici/README.md
//
// A message supports encoding key-value pairs, lists, and sub-sections (or sub-messages).
// Within a Message, each value, list, and sub-section is keyed by a string.
//
// The value in a key-value pair is represented by a string, lists are represented by a string slice,
// and sub-sections are represented by *Message. When constructing a Message, other types may be used
// for convenience, and may have rules on how they are converted to an appropriate internal message
// element type. See Message.Set and MarshalMessage for details.
type Message struct {
	// Packet header. Set only for reading and writing message packets.
	header *header

	keys []string
	data map[string]any
}

type header struct {
	ptype uint8
	name  string
	seq   uint64
}

// NewMessage returns an empty Message.
func NewMessage() *Message {
	return &Message{
		header: nil,
		keys:   make([]string, 0),
		data:   make(map[string]any),
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
func MarshalMessage(v any) (*Message, error) {
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
func UnmarshalMessage(m *Message, v any) error {
	return m.unmarshal(v)
}

// Set sets key to value. An error is returned if the underlying type of
// v is not supported.
//
// If the type of v is supported, it is represented in the message as either a
// string, []string, or *Message. The currently supported types are:
//
//   - string
//   - integer types (converted to string)
//   - bool (where true and false are converted to the strings "yes" and "no", respectively)
//   - []string
//   - *Message
//   - map (the map must be valid as per MarshalMessage)
//   - struct (the struct must be valid as per MarshalMessage)
//
// Pointer types of the above are allowed and can be used to differentiate between
// an unset value and a zero value. If a pointer is nil, it is not added to the message.
//
// If the key already exists the value is overwritten, but the ordering
// of the message is not changed.
func (m *Message) Set(key string, value any) error {
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

// Get returns the value of the field identified by key, if it exists. If
// the field does not exist, nil is returned.
//
// The value returned by Get is the internal message representation of that
// field, which means the type is either string, []string, or *Message.
func (m *Message) Get(key string) any {
	v, ok := m.data[key]
	if !ok {
		return nil
	}

	return v
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
			return fmt.Errorf("%w: %v", errCommandFailed, m.data["errmsg"])
		}
	}

	return nil
}

func (m *Message) stringIndent(prefix, indent string) string {
	var str string

	for k, v := range m.elements() {
		switch v := v.(type) {
		case string:
			str += fmt.Sprintf("%s%s%s = %s\n", prefix, indent, k, v)
		case []string:
			str += fmt.Sprintf("%s%s%s = %s\n", prefix, indent, k, strings.Join(v, ","))
		case *Message:
			str += fmt.Sprintf("%s%s%s %s", prefix, indent, k, v.stringIndent(prefix+indent, indent))
		}
	}
	str = fmt.Sprintf("{\n%s%s}\n", str, prefix)

	return str
}

// String returns the string form of m. For readability, the output format is similar to
// swanctl.conf configuration format.
func (m *Message) String() string {
	return m.stringIndent("", "  ")
}

// packetIsNamed returns a bool indicating the packet is a named type
func (m *Message) packetIsNamed() bool {
	if m.header == nil {
		return false
	}

	switch m.header.ptype {
	case /* Named packet types */
		pktCmdRequest,
		pktEventRegister,
		pktEventUnregister,
		pktEvent:

		return true

	case /* Un-named packet types */
		pktCmdResponse,
		pktCmdUnknown,
		pktEventConfirm,
		pktEventUnknown:

		return false
	}

	return false
}

// packetIsValid checks a packet header to make sure it is valid
func (m *Message) packetIsValid() bool {
	if m.header == nil {
		return false
	}

	if m.header.ptype >= pktInvalid {
		return false
	}

	if m.packetIsNamed() && m.header.name == "" {
		return false
	}

	return true
}

func (m *Message) packetIsRequest() bool {
	if m.header == nil {
		return false
	}

	switch m.header.ptype {
	case /* Valid client requests */
		pktCmdRequest,
		pktEventRegister,
		pktEventUnregister:

		return true

	default:
		return false
	}
}

func (m *Message) addItemFull(key string, value any, unique bool) error {
	// Check if the key is already set in the message
	_, exists := m.data[key]

	if exists && unique {
		return fmt.Errorf("key %v already exists in message", key)
	}

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

func (m *Message) addItemUnique(key string, value any) error {
	return m.addItemFull(key, value, true)
}

func (m *Message) addItem(key string, value any) error {
	return m.addItemFull(key, value, false)
}

func (m *Message) elements() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		if m.keys == nil || m.data == nil {
			return
		}

		for _, k := range m.keys {
			if !yield(k, m.data[k]) {
				return
			}
		}
	}
}

func safePutUint8(buf *bytes.Buffer, val int) error {
	limit := ^uint8(0)

	if int64(val) > int64(limit) {
		return fmt.Errorf("val too long (%d > %d)", val, limit)
	}

	// We can safely convert now, because we just checked that it will not overflow.
	// #nosec G115
	if err := buf.WriteByte(uint8(val)); err != nil {
		return err
	}

	return nil
}

func safePutUint16(buf *bytes.Buffer, val int) error {
	limit := ^uint16(0)
	b := make([]byte, 2)

	if int64(val) > int64(limit) {
		return fmt.Errorf("val too long (%d > %d)", val, limit)
	}

	// We can safely convert now, because we just checked that it will not overflow.
	binary.BigEndian.PutUint16(b, uint16(val)) // #nosec G115

	if _, err := buf.Write(b); err != nil {
		return err
	}

	return nil
}

func safePutUint32(buf *bytes.Buffer, val int) error {
	limit := ^uint32(0)
	b := make([]byte, 4)

	if int64(val) > int64(limit) {
		return fmt.Errorf("val too long (%d > %d)", val, limit)
	}

	// We can safely convert now, because we just checked that it will not overflow.
	binary.BigEndian.PutUint32(b, uint32(val)) // #nosec G115

	if _, err := buf.Write(b); err != nil {
		return err
	}

	return nil
}

func decodeKey(buf *bytes.Buffer) (string, error) {
	// Read the key from the buffer
	n, err := buf.ReadByte()
	if err != nil {
		return "", fmt.Errorf("%v: %v", errDecoding, err)
	}
	if n == 0 {
		return "", fmt.Errorf("%v: key cannot be empty", errDecoding)
	}

	k := string(buf.Next(int(n)))
	if len(k) != int(n) {
		return "", errBadKey
	}

	return k, nil
}

func decodeValue(buf *bytes.Buffer) (string, error) {
	// Read the value's length
	n := buf.Next(2)
	if len(n) != 2 {
		return "", errEndOfBuffer
	}

	// Read the value from the buffer
	vl := int(binary.BigEndian.Uint16(n))
	v := string(buf.Next(vl))

	if len(v) != vl {
		return "", errBadValue
	}

	return v, nil
}

func encodeKey(buf *bytes.Buffer, key string) error {
	if key == "" {
		return fmt.Errorf("%v: cannot encode empty key", errEncoding)
	}

	// Write the key length and key
	if err := safePutUint8(buf, len(key)); err != nil {
		return fmt.Errorf("%v: %v", errEncoding, err)
	}

	if _, err := buf.WriteString(key); err != nil {
		return fmt.Errorf("%v: %v", errEncoding, err)
	}

	return nil
}

func encodeValue(buf *bytes.Buffer, value string) error {
	// Write the value's length to the buffer as two bytes
	if err := safePutUint16(buf, len(value)); err != nil {
		return fmt.Errorf("%v: %v", errEncoding, err)
	}

	// Write the value to the buffer
	if _, err := buf.WriteString(value); err != nil {
		return fmt.Errorf("%v: %v", errEncoding, err)
	}

	return nil
}

func (m *Message) encode() ([]byte, error) {
	buf := bytes.NewBuffer([]byte{})

	if !m.packetIsValid() {
		return nil, fmt.Errorf("%v: cannot encode invalid packet", errEncoding)
	}

	if err := buf.WriteByte(m.header.ptype); err != nil {
		return nil, fmt.Errorf("%v: %v", errEncoding, err)
	}

	if m.packetIsNamed() {
		if err := encodeKey(buf, m.header.name); err != nil {
			return nil, err
		}
	}

	if err := m.encodeElements(buf); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (m *Message) decode(data []byte) error {
	m.header = &header{}
	buf := bytes.NewBuffer(data)

	// Parse the message header first.
	b, err := buf.ReadByte()
	if err != nil {
		return fmt.Errorf("%v: %v", errDecoding, err)
	}
	if b >= pktInvalid {
		return fmt.Errorf("%v: invalid packet type %v", errDecoding, b)
	}
	m.header.ptype = b

	if m.packetIsNamed() {
		name, err := decodeKey(buf)
		if err != nil {
			return err
		}
		m.header.name = name
	}

	for buf.Len() > 0 {
		b, err = buf.ReadByte()
		if err != nil && err != io.EOF {
			return fmt.Errorf("%v: %v", errDecoding, err)
		}

		// Determine the next message element
		switch b {
		case msgKeyValue:
			if err := m.decodeKeyValue(buf); err != nil {
				return err
			}

		case msgListStart:
			if err := m.decodeList(buf); err != nil {
				return err
			}

		case msgSectionStart:
			if err := m.decodeSection(buf); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%v: invalid byte %v looking for next element type", errDecoding, b)
		}
	}

	return nil
}

// encodeElements encodes all of the message elements to the buffer.
func (m *Message) encodeElements(buf *bytes.Buffer) error {
	for k, v := range m.elements() {
		switch v := v.(type) {
		case string:
			if err := encodeKeyValue(buf, k, v); err != nil {
				return err
			}

		case []string:
			if err := encodeList(buf, k, v); err != nil {
				return err
			}

		case *Message:
			if err := encodeSection(buf, k, v); err != nil {
				return err
			}

		default:
			// This should never happen.
			return errUnsupportedType
		}
	}

	return nil
}

// encodeKeyValue will return a byte slice of an encoded key-value pair.
//
// The size of the byte slice is the length of the key and value, plus four bytes:
// one byte for message element type, one byte for key length, and two bytes for value
// length.
func encodeKeyValue(buf *bytes.Buffer, key string, value string) error {
	// Initialize buffer to indictate the message element type
	// is a key-value pair
	if err := buf.WriteByte(msgKeyValue); err != nil {
		return err
	}

	if err := encodeKey(buf, key); err != nil {
		return err
	}

	if err := encodeValue(buf, value); err != nil {
		return err
	}

	return nil
}

// encodeList will encode the list to the provided buffer.
//
// The size of the byte slice is the length of the key and total length of
// the list (sum of length of the items in the list), plus three bytes for each
// list item: one for message element type, and two for item length. Another three
// bytes are used to indicate list start and list stop, and the length of the key.
func encodeList(buf *bytes.Buffer, key string, list []string) error {
	// Initialize buffer to indictate the message element type
	// is the start of a list
	if err := buf.WriteByte(msgListStart); err != nil {
		return err
	}

	if err := encodeKey(buf, key); err != nil {
		return err
	}

	for _, item := range list {
		// Indicate that this is a list item
		if err := buf.WriteByte(msgListItem); err != nil {
			return fmt.Errorf("%v: %v", errEncoding, err)
		}

		if err := encodeValue(buf, item); err != nil {
			return err
		}
	}

	// Indicate the end of the list
	if err := buf.WriteByte(msgListEnd); err != nil {
		return fmt.Errorf("%v: %v", errEncoding, err)
	}

	return nil
}

// encodeSection will encode the section to the given buffer.
func encodeSection(buf *bytes.Buffer, key string, section *Message) error {
	// Initialize buffer to indictate the message element type
	// is the start of a section
	if err := buf.WriteByte(msgSectionStart); err != nil {
		return err
	}

	if err := encodeKey(buf, key); err != nil {
		return err
	}

	if err := section.encodeElements(buf); err != nil {
		return err
	}

	// Indicate the end of the section
	if err := buf.WriteByte(msgSectionEnd); err != nil {
		return fmt.Errorf("%v: %v", errEncoding, err)
	}

	return nil
}

// decodeKeyValue will decode a key-value pair and write it to the message's
// data.
func (m *Message) decodeKeyValue(buf *bytes.Buffer) error {
	key, err := decodeKey(buf)
	if err != nil {
		return err
	}

	value, err := decodeValue(buf)
	if err != nil {
		return err
	}

	if err := m.addItemUnique(key, value); err != nil {
		return fmt.Errorf("%v: %v", errDecoding, err)
	}

	return nil
}

// decodeList will decode a list and write it to the message's data.
func (m *Message) decodeList(buf *bytes.Buffer) error {
	var list []string

	key, err := decodeKey(buf)
	if err != nil {
		return err
	}

	b, err := buf.ReadByte()
	if err != nil {
		return fmt.Errorf("%v: %v", errDecoding, err)
	}

	// Read the list from the buffer
	for b != msgListEnd {
		// Ensure this is the beginning of a list item
		if b != msgListItem {
			return errExpectedBeginning
		}

		value, err := decodeValue(buf)
		if err != nil {
			return err
		}

		list = append(list, value)

		b, err = buf.ReadByte()
		if err != nil {
			return fmt.Errorf("%v: %v", errDecoding, err)
		}
	}

	if err := m.addItemUnique(key, list); err != nil {
		return fmt.Errorf("%v: %v", errDecoding, err)
	}

	return nil
}

// decodeSection will decode a section into a message's data.
func (m *Message) decodeSection(buf *bytes.Buffer) error {
	section := NewMessage()

	key, err := decodeKey(buf)
	if err != nil {
		return err
	}

	b, err := buf.ReadByte()
	if err != nil {
		return fmt.Errorf("%v: %v", errDecoding, err)
	}

	for b != msgSectionEnd {
		// Determine the next message element
		switch b {
		case msgKeyValue:
			if err := section.decodeKeyValue(buf); err != nil {
				return err
			}

		case msgListStart:
			if err := section.decodeList(buf); err != nil {
				return err
			}

		case msgSectionStart:
			if err := section.decodeSection(buf); err != nil {
				return err
			}

		default:
			return errExpectedBeginning
		}

		b, err = buf.ReadByte()
		if err != nil {
			return fmt.Errorf("%v: %v", errDecoding, err)
		}
	}

	if err := m.addItemUnique(key, section); err != nil {
		return err
	}

	return nil
}

// sendmsg is a helper to write a message to a given net.Conn.
func sendmsg(conn net.Conn, p *Message) error {
	b, err := p.encode()
	if err != nil {
		return err
	}

	// The packet length must fit in four bytes.
	if uint64(len(b)) > uint64(^uint32(0)) {
		return fmt.Errorf("packet length (%d) exceeds 4 bytes", len(b))
	}

	raw := make([]byte, 4)
	binary.BigEndian.PutUint32(raw, uint32(len(b))) // #nosec G115
	raw = append(raw, b...)

	if _, err := conn.Write(raw); err != nil {
		return err
	}

	return nil
}

// readmsg is a helper to read a message from a given net.Conn.
func readmsg(conn net.Conn) (*Message, error) {
	raw := make([]byte, 4 /* header length */)
	if _, err := io.ReadFull(conn, raw); err != nil {
		return nil, err
	}

	raw = make([]byte, binary.BigEndian.Uint32(raw))
	if _, err := io.ReadFull(conn, raw); err != nil {
		return nil, err
	}

	p := NewMessage()
	if err := p.decode(raw); err != nil {
		return nil, err
	}

	return p, nil
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

func (m *Message) marshal(v any) error {
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
				return fmt.Errorf("%v: cannot marshal non-struct inlined field %v", errMarshalUnsupportedType, rfv.Kind())
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

func (m *Message) unmarshal(v any) error {
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
				return fmt.Errorf("%v: cannot unmarshal into non-struct inlined field %v", errUnmarshalUnsupportedType, rfv.Kind())
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

	for k, v := range m.elements() {
		key := reflect.ValueOf(k)
		val := reflect.ValueOf(v)

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
