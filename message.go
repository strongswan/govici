// Package vici implements a strongSwan vici protocol client
package vici

// MessageElementType represents a message element type
type MessageElementType uint8

const (
	// Begin a new section having a name
	SectionStart MessageElementType = iota + 1

	// End a previously started section
	SectionEnd

	// Define a value for a named key in the current section
	KeyValue

	// Begin a name list for list items
	ListStart

	// Dfeine an unnamed item value in the current list
	ListItem

	// End a prevsiously started list
	ListEnd
)
