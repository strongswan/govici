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
		t.Errorf("Unmarshalled boolean value is invalid.\nExpected: true\nReceived: %+v", boolMessage.Field)
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
		t.Errorf("Unmarshalled boolean value is invalid.\nExpected: false\nReceived: %+v", boolMessage.Field)
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
		t.Errorf("Unmarshalled int value is invalid.\nExpected: 23\nReceived: %+v", intMessage.Field)
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
		t.Errorf("Unmarshalled int value is invalid.\nExpected: -23\nReceived: %+v", intMessage.Field)
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
		t.Errorf("Unmarshalled int8 value is invalid.\nExpected: 23\nReceived: %+v", intMessage.Field)
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
		t.Errorf("Unmarshalled int8 value is invalid.\nExpected: -23 (Overflow)\nReceived: %+v", intMessage.Field)
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
		t.Errorf("Unmarshalled uint value is invalid.\nExpected: 23\nReceived: %+v", intMessage.Field)
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
