package main

import (
	"fmt"

	"github.com/enr0n/vici"
)

func main() {
	session, err := vici.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}

	events := []string{vici.IKEUpdown, vici.ChildUpdown}
	go session.Listen(events)

	for {
		m, err := session.NextEvent()
		if err != nil {
			fmt.Print(err)
			continue
		}

		fmt.Println(m)
	}
}
