package main

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/strongswan/govici/vici"
)

type cert struct {
	Type string `vici:"type"`
	Flag string `vici:"flag"`
	Data string `vici:"data"`
}

func loadX509Cert(path string, cacert bool) error {
	s, err := vici.NewSession()
	if err != nil {
		return err
	}
	defer s.Close()

	flag := "NONE"
	if cacert {
		flag = "CA"
	}

	// Read cert data from the file
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(data)

	cert := cert{
		Type: "X509",
		Flag: flag,
		Data: string(block.Bytes),
	}

	m, err := vici.MarshalMessage(&cert)
	if err != nil {
		return err
	}

	_, err = s.CommandRequest("load-cert", m)

	return err
}

func main() {
	if err := loadX509Cert("/path/to/cert", false); err != nil {
		fmt.Println(err)
	}
}
