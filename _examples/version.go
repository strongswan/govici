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

	m, err := session.Version()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%v %v\n", m.Get("daemon"), m.Get("version"))
}
