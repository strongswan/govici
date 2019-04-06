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

	conns, err := session.GetConns()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, conn := range conns {
		fmt.Println(conn)
	}
}
