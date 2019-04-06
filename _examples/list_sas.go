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

	sas, err := session.ListSAs(nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	// ListSAs returns a message stream of list-sa events
	for _, m := range sas.Messages() {
		fmt.Println(m)
	}
}
