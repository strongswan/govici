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

	m, err := session.CommandRequest("version", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, k := range m.Keys() {
		fmt.Printf("%v: %v\n", k, m.Get(k))
	}
}
