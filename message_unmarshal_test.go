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
		t.Errorf("Error unmarshalling bool value: %v", err)
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
		t.Errorf("Error unmarshalling bool value: %v", err)
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
