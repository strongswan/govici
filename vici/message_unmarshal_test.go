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
	"testing"
)

func TestUnmarshalBoolTrue(t *testing.T) {
	boolMessage := struct {
		Field bool `vici:"field"`
	}{
		Field: false,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "yes",
		},
	}

	err := UnmarshalMessage(m, &boolMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling bool value: %v", err)
	}

	if boolMessage.Field != true {
		t.Fatalf("Unmarshalled boolean value is invalid.\nExpected: true\nReceived: %+v", boolMessage.Field)
	}
}

func TestUnmarshalBoolFalse(t *testing.T) {
	boolMessage := struct {
		Field bool `vici:"field"`
	}{
		Field: true,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "no",
		},
	}

	err := UnmarshalMessage(m, &boolMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling bool value: %v", err)
	}

	if boolMessage.Field != false {
		t.Fatalf("Unmarshalled boolean value is invalid.\nExpected: false\nReceived: %+v", boolMessage.Field)
	}
}

func TestUnmarshalBoolInvalid(t *testing.T) {
	boolMessage := struct {
		Field bool `vici:"field"`
	}{
		Field: true,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "invalid-not-a-bool",
		},
	}

	err := UnmarshalMessage(m, &boolMessage)
	if err == nil {
		t.Error("Expected error when unmarshalling invalid boolean value. None was returned.")
	}
}

func TestUnmarshalBoolTruePtr(t *testing.T) {
	boolMessage := struct {
		Field *bool `vici:"field"`
	}{
		Field: nil,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "yes",
		},
	}

	err := UnmarshalMessage(m, &boolMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling bool value to pointer: %v", err)
	}

	if boolMessage.Field == nil {
		t.Fatalf("Unmarshalled boolean pointer is nil.")
	}

	if *boolMessage.Field != true {
		t.Fatalf("Unmarshalled boolean value is invalid.\nExpected: true\nReceived: %+v", *boolMessage.Field)
	}
}

func TestUnmarshalBoolFalsePtr(t *testing.T) {
	boolMessage := struct {
		Field *bool `vici:"field"`
	}{
		Field: nil,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "no",
		},
	}

	err := UnmarshalMessage(m, &boolMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling bool value to pointer: %v", err)
	}

	if boolMessage.Field == nil {
		t.Fatalf("Unmarshalled boolean pointer is nil.")
	}

	if *boolMessage.Field != false {
		t.Fatalf("Unmarshalled boolean value is invalid.\nExpected: false\nReceived: %+v", *boolMessage.Field)
	}
}

func TestUnmarshalInt(t *testing.T) {
	intMessage := struct {
		Field int `vici:"field"`
	}{
		Field: 0,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "23",
		},
	}

	err := UnmarshalMessage(m, &intMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling int value: %v", err)
	}

	if intMessage.Field != 23 {
		t.Fatalf("Unmarshalled int value is invalid.\nExpected: 23\nReceived: %+v", intMessage.Field)
	}
}

func TestUnmarshalInt2(t *testing.T) {
	intMessage := struct {
		Field int `vici:"field"`
	}{
		Field: 0,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "-23",
		},
	}

	err := UnmarshalMessage(m, &intMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling int value: %v", err)
	}

	if intMessage.Field != -23 {
		t.Fatalf("Unmarshalled int value is invalid.\nExpected: -23\nReceived: %+v", intMessage.Field)
	}
}

func TestUnmarshalInt8(t *testing.T) {
	intMessage := struct {
		Field int8 `vici:"field"`
	}{
		Field: 0,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "23",
		},
	}

	err := UnmarshalMessage(m, &intMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling int8 value: %v", err)
	}

	if intMessage.Field != 23 {
		t.Fatalf("Unmarshalled int8 value is invalid.\nExpected: 23\nReceived: %+v", intMessage.Field)
	}
}

func TestUnmarshalInt8Overflow(t *testing.T) {
	intMessage := struct {
		Field int8 `vici:"field"`
	}{
		Field: 0,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "1001",
		},
	}

	err := UnmarshalMessage(m, &intMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling int8 value: %v", err)
	}

	if intMessage.Field == 23 {
		t.Fatalf("Unmarshalled int8 value is invalid.\nExpected: -23 (Overflow)\nReceived: %+v", intMessage.Field)
	}
}

func TestUnmarshalUint(t *testing.T) {
	intMessage := struct {
		Field uint `vici:"field"`
	}{
		Field: 0,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "23",
		},
	}

	err := UnmarshalMessage(m, &intMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling uint value: %v", err)
	}

	if intMessage.Field != 23 {
		t.Fatalf("Unmarshalled uint value is invalid.\nExpected: 23\nReceived: %+v", intMessage.Field)
	}
}

func TestUnmarshalUintInvalid(t *testing.T) {
	intMessage := struct {
		Field uint `vici:"field"`
	}{
		Field: 0,
	}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "-1",
		},
	}

	err := UnmarshalMessage(m, &intMessage)
	if err == nil {
		t.Error("Expected error when unmarshalling invalid uint value. None was returned.")
	}
}

func TestUnmarshalEnumType(t *testing.T) {
	type TestType string
	const testValue TestType = "test-value"

	enumMessage := struct {
		Field TestType `vici:"field"`
	}{}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "test-value",
		},
	}

	err := UnmarshalMessage(m, &enumMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling enum type value: %v", err)
	}

	if enumMessage.Field != testValue {
		t.Fatalf("Unmarshalled uint value is invalid.\nExpected: %+v\nReceived: %+v", testValue, enumMessage.Field)
	}
}

func TestUnmarshalEmbeddedStruct(t *testing.T) {
	const testValue = "unmarshalled-embedded-value"

	type Embedded struct {
		Field string `vici:"field"`
	}

	embeddedMessage := struct {
		Embedded `vici:"embedded"`
	}{}

	m := &Message{
		[]string{"embedded"},
		map[string]interface{}{
			"embedded": &Message{
				[]string{"field"},
				map[string]interface{}{
					"field": testValue,
				},
			},
		},
	}

	err := UnmarshalMessage(m, &embeddedMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling into embedded struct: %v", err)
	}

	if embeddedMessage.Field != testValue {
		t.Fatalf("Unmarshalled embedded value is invalid.\nExpected: %+v\nReceived: %+v", testValue, embeddedMessage.Field)
	}
}

func TestUnmarshalInline(t *testing.T) {
	testValue := "unmarshal-inline"

	type Embedded struct {
		Field string `vici:"field"`
	}

	inlineMessage := struct {
		Embedded `vici:",inline"`
	}{}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": testValue,
		},
	}

	err := UnmarshalMessage(m, &inlineMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling into inlined embedded struct: %v", err)
	}

	if inlineMessage.Field != testValue {
		t.Fatalf("Unmarshalled inlined embedded value is invalid.\nExpected: %+v\nReceived: %+v", testValue, inlineMessage.Field)
	}
}

func TestUnmarshalInlineInvalidType(t *testing.T) {
	inlineMessage := struct {
		Field string `vici:",inline"`
	}{}

	m := &Message{
		[]string{"field"},
		map[string]interface{}{
			"field": "test-value",
		},
	}

	err := UnmarshalMessage(m, &inlineMessage)
	if err == nil {
		t.Error("Expected error when unmarshalling invalid inlined embedded type. None was returned.")
	}
}

func TestUnmarshalInlineComposite(t *testing.T) {
	testValue := "unmarshal-inline-composite"
	otherValue := "other-value"

	type Embedded struct {
		Field string `vici:"field"`
	}

	inlineMessage := struct {
		Embedded `vici:",inline"`
		Other    string `vici:"other"`
	}{}

	m := &Message{
		[]string{"field", "other"},
		map[string]interface{}{
			"field": testValue,
			"other": otherValue,
		},
	}

	err := UnmarshalMessage(m, &inlineMessage)
	if err != nil {
		t.Fatalf("Error unmarshalling into inlined embedded struct: %v", err)
	}

	if inlineMessage.Field != testValue {
		t.Fatalf("Unmarshalled inlined embedded value is invalid.\nExpected: %+v\nReceived: %+v", testValue, inlineMessage.Field)
	}

	if inlineMessage.Other != otherValue {
		t.Fatalf("Unmarshalled inlined embedded value is invalid.\nExpected: %+v\nReceived: %+v", otherValue, inlineMessage.Other)
	}
}
