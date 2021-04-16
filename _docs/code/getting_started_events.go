package main

import (
	"context"
	"fmt"

	"github.com/strongswan/govici/vici"
)

func main() {
	session, err := vici.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer session.Close()

	// Subscribe to 'ike-updown' and 'log' events.
	if err := session.Subscribe("ike-updown", "log"); err != nil {
		fmt.Println(err)
		return
	}

	// IKE SA configuration name
	name := "rw"

	for {
		e, err := session.NextEvent(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}

		// The Event.Name field corresponds to the event name
		// we used to make the subscription. The Event.Message
		// field contains the Message from the server.
		switch e.Name {
		case "ike-updown":
			m, ok := e.Message.Get(name).(*vici.Message)
			if !ok {
				fmt.Printf("Expected *Message in field 'name', but got %T", m)
				continue
			}

			state := m.Get("state")
			fmt.Printf("IKE SA state changed (name=%s): %s\n", name, state)
		case "log":
			// Log events contain a 'msg' field with the log message
			fmt.Println(e.Message.Get("msg"))
		}
	}
}
