package main

import (
	"fmt"

	"github.com/enr0n/vici"
)

type initiateOptions struct {
	child      string `vici:"child,omitempty"`
	ike        string `vici:"ike,omitempty"`
	timeout    string `vici:"timeout,omitempty"`
	initLimits string `vici:"init-limits,omitempty"`
	logLevel   string `vici:"loglevel,omitempty"`
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
