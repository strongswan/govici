package main

import (
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

	m := vici.NewMessage()

	if err := m.Set("flag", "CA"); err != nil {
		fmt.Println(err)
		return
	}

	ms, err := session.StreamedCommandRequest("list-certs", "list-cert", m)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, m := range ms.Messages() {
		if m.Err() != nil {
			fmt.Println(err)
			return
		}

		// Process CA cert information
		// ...
	}
}
