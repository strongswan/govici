# govici

[![Build Status](https://travis-ci.org/strongswan/govici.svg?branch=master)](https://travis-ci.org/strongswan/govici)
[![GoDoc](https://godoc.org/github.com/strongswan/govici?status.svg)](https://godoc.org/github.com/strongswan/govici)
[![Go Report Card](https://goreportcard.com/badge/github.com/strongswan/govici)](https://goreportcard.com/report/github.com/strongswan/govici)

## About

The strongSwan [vici protocol](https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html) is used for external applications to monitor, configure, and control the IKE daemon charon. This Go package provides a pure-go implementation of a vici client library.

The package documentation can be found on [godoc](https://godoc.org/github.com/strongswan/govici).

## Getting started
`go get -u github.com/strongswan/govici`

## Examples

Below are some examples of how to use `vici.Session`. For a complete list of supported commands, message parameters, and event types, see the vici [README](https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html).

### Command Requests

#### version

This example shows how a command request can be made to get version information about the running charon daemon. The first argument to `CommandRequest` is the command name. No additional arguments are required, so `nil` is given. Iterating over the response `Message`'s keys will give all information returned by the daemon.

```go
package main

import (
	"fmt"
	"github.com/strongswan/govici"
)

func main() {
	session, err := vici.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}

	m, err := session.CommandRequest("version", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

        for _, k := range m.Keys() {
                fmt.Printf("%v: %v\n", k, m.Get(k))
        }
}
```

#### get-conns

This example shows how `CommandRequest` can be used to get a list of connection names that have been loaded over vici. Note that `Get` returns `interface{}`, so a type assertion is needed to  iterate over the items returned in the `"conns"` field.

```go
package main

import (
	"fmt"
	"github.com/strongswan/govici"
)

func main() {
	session, err := vici.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}

	m, err := session.CommandRequest("get-conns", nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	conns := m.Get("conns")
	if conns == nil {
		fmt.Println("Expected connections field in message")
		return
	}

	for _, conn := range conns.([]string) {
		fmt.Println(conn)
	}
}
```

#### install

Use `Set` to populate a `Message` with the arguments needed for a command. Here, the command `"install"` accepts a child SA name and an optional IKE SA name to find the child under. The success of the command can be checked using `Err`.

```go
package main

import (
	"fmt"
	"github.com/strongswan/govici"
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

	resp, err := session.CommandRequest("install", m)
	if err != nil {
		fmt.Println(err)
		return
	}

        if resp.Err() != nil {
                fmt.Println("Command failed:", err)
        }

        fmt.Println("Command succeeded")
}
```

### Streamed Command Requests

#### initiate

This example shows how to use `StreamedCommandRequest`. The command `"initiate"` initiates an SA while streaming `"control-log"` events. A `MessageStream` is returned after all messages have been received, and the session has stopped listening for `"control-log"` events (unless otherwise specified using `Listen`).

This also shows how `MarshalMessage` can be used to construct a `Message` from a struct by using struct tags.

```go
package main

import (
	"fmt"
	"github.com/strongswan/govici"
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

	ms, err := session.StreamedCommandRequest("initiate", "control-log", m)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, m := range ms.Messages() {
		fmt.Println(m)
	}

	fmt.Println("Initiated SA")
}
```

### Event Listener

This example shows a session that registers for `"ike-updown"` and `"child-updown"` events. `Listen` does not return unless the event channel is closed, so it should be run in another goroutine. Events received by the listener are given by `NextEvent`. If there is no event in the buffer at the time  `NextEvent` is called, it will block until an event is received.

```go
package main

import (
	"fmt"
	"github.com/strongswan/govici"
)

func main() {
	session, err := vici.NewSession()
	if err != nil {
		fmt.Println(err)
		return
	}

	events := []string{"ike-updown", "child-updown"}
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
```
