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
		t.Errorf("Marshalled boolean value is invalid.\nExpected: yes\nReceived: %+v", value)
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
		t.Errorf("Marshalled boolean value is invalid.\nExpected: no\nReceived: %+v", value)
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
		t.Errorf("Marshalled int value is invalid.\nExpected: 23\nReceived: %+v", value)
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
		t.Errorf("Marshalled int value is invalid.\nExpected: -23\nReceived: %+v", value)
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
		t.Errorf("Marshalled int8 value is invalid.\nExpected: 23\nReceived: %+v", value)
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
		t.Errorf("Marshalled uint value is invalid.\nExpected: 23\nReceived: %+v", value)
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
		t.Errorf("Marshalled uint8 value is invalid.\nExpected: 23\nReceived: %+v", value)
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
		t.Errorf("Marshalled enum type value is invalid.\nExpected: %+v\nReceived: %+v", testValue, value)
	}

}
