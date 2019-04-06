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

	m := vici.NewMessage()
	err = m.Set("child", "child_sa_name")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = m.Set("ike", "ike_sa_name")
	if err != nil {
		fmt.Println(err)
		return
	}

	err = session.Install(m)
	if err != nil {
		fmt.Println(err)
		return
	}
}
