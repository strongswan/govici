package main

import (
	"fmt"

	"github.com/enr0n/vici"
)

type initiateOptions struct {
	child      string `vici:"child"`
	ike        string `vici:"ike"`
	timeout    string `vici:"timeout"`
	initLimits string `vici:"init-limits"`
	logLevel   string `vici:"loglevel"`
}

func main() {
	session, err := vici.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}

	initOpts := initiateOptions{}

	// Populate struct
	//
	// 	...
	//

	m, err := vici.MarshalMessage(initOpts)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = session.Initiate(m)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Initiated SA")
}
