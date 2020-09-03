// Copyright (C) 2019 Arroyo Networks, Inc
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
	"reflect"
	"testing"
)

func TestMarshalTagSkip(t *testing.T) {
	skipMessage := struct {
		Skipped string `vici:""`
	}{
		Skipped: "skipped",
	}

	m, err := MarshalMessage(skipMessage)
	if err != nil {
		t.Fatalf("Error marshalling skipped value: %v", err)
	}

	if len(m.Keys()) != 0 {
		t.Fatalf("Marshalled message has keys.\nExpected: 0\nReceived: %d", len(m.Keys()))
	}
}

func TestMarshalTagSkipDash(t *testing.T) {
	skipMessage := struct {
		Skipped string `vici:"-"`
	}{
		Skipped: "skipped",
	}

	m, err := MarshalMessage(skipMessage)
	if err != nil {
		t.Fatalf("Error marshalling skipped value: %v", err)
	}

	if len(m.Keys()) != 0 {
		t.Fatalf("Marshalled message has keys.\nExpected: 0\nReceived: %d", len(m.Keys()))
	}
}

func TestMarshalTagSkipWithOpt(t *testing.T) {
	skipMessage := struct {
		Skipped string `vici:",testOpt"`
	}{
		Skipped: "skipped",
	}

	m, err := MarshalMessage(skipMessage)
	if err != nil {
		t.Fatalf("Error marshalling skipped value: %v", err)
	}

	if len(m.Keys()) != 0 {
		t.Fatalf("Marshalled message has keys.\nExpected: 0\nReceived: %d", len(m.Keys()))
	}
}

func TestMarshalTagSkipDashWithOpt(t *testing.T) {
	skipMessage := struct {
		Skipped string `vici:"-,testOpt"`
	}{
		Skipped: "skipped",
	}

	m, err := MarshalMessage(skipMessage)
	if err != nil {
		t.Fatalf("Error marshalling skipped value: %v", err)
	}

	if len(m.Keys()) != 0 {
		t.Fatalf("Marshalled message has keys.\nExpected: 0\nReceived: %d", len(m.Keys()))
	}
}

func TestMarshalBoolTrue(t *testing.T) {
	boolMessage := struct {
		Field bool `vici:"field"`
	}{
		Field: true,
	}

	m, err := MarshalMessage(boolMessage)
	if err != nil {
		t.Fatalf("Error marshalling bool value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "yes") {
		t.Fatalf("Marshalled boolean value is invalid.\nExpected: yes\nReceived: %+v", value)
	}
}

func TestMarshalBoolFalse(t *testing.T) {
	boolMessage := struct {
		Field bool `vici:"field"`
	}{
		Field: false,
	}

	m, err := MarshalMessage(boolMessage)
	if err != nil {
		t.Fatalf("Error marshalling bool value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "no") {
		t.Fatalf("Marshalled boolean value is invalid.\nExpected: no\nReceived: %+v", value)
	}
}

func TestMarshalBoolTruePtr(t *testing.T) {
	boolValue := true
	boolMessage := struct {
		Field *bool `vici:"field"`
	}{
		Field: &boolValue,
	}

	m, err := MarshalMessage(boolMessage)
	if err != nil {
		t.Fatalf("Error marshalling pointer to bool value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "yes") {
		t.Fatalf("Marshalled boolean pointer value is invalid.\nExpected: yes\nReceived: %+v", value)
	}
}

func TestMarshalBoolFalsePtr(t *testing.T) {
	boolValue := false
	boolMessage := struct {
		Field *bool `vici:"field"`
	}{
		Field: &boolValue,
	}

	m, err := MarshalMessage(boolMessage)
	if err != nil {
		t.Fatalf("Error marshalling pointer to bool value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "no") {
		t.Fatalf("Marshalled boolean pointer value is invalid.\nExpected: no\nReceived: %+v", value)
	}
}

func TestMarshalBoolNilPtr(t *testing.T) {
	boolMessage := struct {
		Field *bool `vici:"field"`
	}{
		Field: nil,
	}

	m, err := MarshalMessage(boolMessage)
	if err != nil {
		t.Fatalf("Error marshalling pointer to bool value: %v", err)
	}

	value := m.Get("field")
	if value != nil {
		t.Fatalf("Marshalled nil boolean pointer value is present.\nReceived: %+v", value)
	}
}

func TestMarshalInt(t *testing.T) {
	intMessage := struct {
		Field int `vici:"field"`
	}{
		Field: 23,
	}

	m, err := MarshalMessage(intMessage)
	if err != nil {
		t.Fatalf("Error marshalling int value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "23") {
		t.Fatalf("Marshalled int value is invalid.\nExpected: 23\nReceived: %+v", value)
	}
}

func TestMarshalInt2(t *testing.T) {
	intMessage := struct {
		Field int `vici:"field"`
	}{
		Field: -23,
	}

	m, err := MarshalMessage(intMessage)
	if err != nil {
		t.Fatalf("Error marshalling int value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "-23") {
		t.Fatalf("Marshalled int value is invalid.\nExpected: -23\nReceived: %+v", value)
	}
}

func TestMarshalInt8(t *testing.T) {
	intMessage := struct {
		Field int8 `vici:"field"`
	}{
		Field: 23,
	}

	m, err := MarshalMessage(intMessage)
	if err != nil {
		t.Fatalf("Error marshalling int8 value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "23") {
		t.Fatalf("Marshalled int8 value is invalid.\nExpected: 23\nReceived: %+v", value)
	}
}

func TestMarshalUint(t *testing.T) {
	intMessage := struct {
		Field uint `vici:"field"`
	}{
		Field: 23,
	}

	m, err := MarshalMessage(intMessage)
	if err != nil {
		t.Fatalf("Error marshalling uint value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "23") {
		t.Fatalf("Marshalled uint value is invalid.\nExpected: 23\nReceived: %+v", value)
	}
}

func TestMarshalUint8(t *testing.T) {
	intMessage := struct {
		Field uint8 `vici:"field"`
	}{
		Field: 23,
	}

	m, err := MarshalMessage(intMessage)
	if err != nil {
		t.Fatalf("Error marshalling uint8 value: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, "23") {
		t.Fatalf("Marshalled uint8 value is invalid.\nExpected: 23\nReceived: %+v", value)
	}
}

func TestMarshalEnumType(t *testing.T) {
	type TestType string
	const testValue TestType = "test-value"

	enumMessage := struct {
		Field TestType `vici:"field"`
	}{
		Field: testValue,
	}

	m, err := MarshalMessage(enumMessage)
	if err != nil {
		t.Fatalf("Error marshalling enum type value: %v", err)
	}

	value := m.Get("field")
	if value.(string) != string(testValue) {
		t.Fatalf("Marshalled enum type value is invalid.\nExpected: %+v\nReceived: %+v", testValue, value)
	}
}

func TestMarshalEmbeddedMap(t *testing.T) {
	mapValue := map[string]interface{}{"sub": goldUnmarshaled}

	mapMessage := struct {
		Field map[string]interface{} `vici:"field"`
	}{
		Field: mapValue,
	}

	m, err := MarshalMessage(mapMessage)
	if err != nil {
		t.Fatalf("Error marshalling map value: %v", err)
	}

	value := m.Get("field")
	field, ok := value.(*Message)
	if !ok {
		t.Fatalf("Embedded map key was not marshaled as a sub-message")
	}

	value = field.Get("sub")
	if !reflect.DeepEqual(value, goldMarshaled) {
		t.Fatalf("Marshalled map value is invalid.\nExpected: %+v\nReceived: %+v", goldMarshaled, value)
	}
}

func TestMarshalEmbeddedStruct(t *testing.T) {
	const testValue = "marshalled-embedded-value"

	type Embedded struct {
		Field string `vici:"field"`
	}

	embeddedMessage := struct {
		Embedded `vici:"embedded"`
	}{}

	embeddedMessage.Field = testValue

	m, err := MarshalMessage(embeddedMessage)
	if err != nil {
		t.Fatalf("Errorf marshalling embedded struct: %v", err)
	}

	value := m.Get("embedded")
	embedded, ok := value.(*Message)
	if !ok {
		t.Fatalf("Embedded struct was not marshalled as a sub-message")
	}

	value = embedded.Get("field")
	if !reflect.DeepEqual(value, testValue) {
		t.Fatalf("Marshalled embedded struct value is invalid.\nExpected: %+v\nReceived: %+v", testValue, value)
	}
}

func TestMarshalInline(t *testing.T) {
	testValue := "marshal-inline"

	type Embedded struct {
		Field string `vici:"field"`
	}

	inlineMessage := struct {
		Embedded `vici:",inline"`
	}{}

	inlineMessage.Field = testValue

	m, err := MarshalMessage(inlineMessage)
	if err != nil {
		t.Fatalf("Error marshalling inlined embedded struct: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, testValue) {
		t.Fatalf("Marshalled inlined embedded value is invalid.\nExpected: %+v\nReceived: %+v", testValue, value)
	}
}

func TestMarshalInlineInvalidType(t *testing.T) {
	inlineMessage := struct {
		Field string `vici:",inline"`
	}{}

	inlineMessage.Field = "inline-value"

	_, err := MarshalMessage(inlineMessage)
	if err == nil {
		t.Error("Expected error when marshalling invalid inlined embedded type. None was returned.")
	}
}

func TestMarshalInlineComposite(t *testing.T) {
	testValue := "marshal-inline-composite"
	otherValue := "other-value"

	type Embedded struct {
		Field string `vici:"field"`
	}

	inlineMessage := struct {
		Embedded `vici:",inline"`
		Other    string `vici:"other"`
	}{}

	inlineMessage.Field = testValue
	inlineMessage.Other = otherValue

	m, err := MarshalMessage(inlineMessage)
	if err != nil {
		t.Fatalf("Error marshalling inlined embedded struct: %v", err)
	}

	value := m.Get("field")
	if !reflect.DeepEqual(value, testValue) {
		t.Fatalf("Marshalled inlined embedded value is invalid.\nExpected: %+v\nReceived: %+v", testValue, value)
	}

	value = m.Get("other")
	if !reflect.DeepEqual(value, otherValue) {
		t.Fatalf("Marshalled inlined embedded value is invalid.\nExpected: %+v\nReceived: %+v", otherValue, value)
	}
}
