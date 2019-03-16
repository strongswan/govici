// Package vici implements a strongSwan vici protocol client
package vici

// PacketType represents the packet type
type PacketType uint8

const (
	// A name request message
	CmdRequest = iota

	// An unnamed response message for a request
	CmdResponse

	// An unnamed response if requested command is unknown
	CmdUnkown

	// A named event registration request
	EventRegister

	// A name event deregistration request
	EventUnregister

	// An unnamed response for successful event (de-)registration
	EventConfirm

	// An unnamed response if event (de-)registration failed
	EventUnknown

	// A named event message
	Event
)
