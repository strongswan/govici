# Getting Started with vici
- [Command Requests](#command-requests)
- [Streamed Command Requests](#streamed-command-requests)
- [Event Listener](#event-listener)
- [Message Marshaling](#message-marshaling)
- [Full Example: Loading and Establishing a Connection](#putting-it-all-together)

## Command Requests

Let's start with a simple example to try and understand how vici works. If you are running strongswan with the charon daemon, and the vici plugin is enabled (the default), you can create a vici client session like this:

```go
s, err := vici.NewSession()
if err != nil {
        fmt.Println(err)
        return
}
defer s.Close() 
```

Say we wanted to get the version information of the charon daemon running on our system. If we look at the [vici README](https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html), we can find the `version` command in the **Client-initiated commands** section. The README gives the following definition of the `version` command's message parameters:

```
{} => {
    daemon = <IKE daemon name>
    version = <strongSwan version>
    sysname = <operating system name>
    release = <operating system release>
    machine = <hardware identifier>
}
```

This means that the command does not accept any arguments, and returns five key-value pairs. So, there is no need to construct a request message for this command. Now all we have to do is make a command request using the `Session.CommandRequest` function.

```go
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
``` 

On my machine, this gives me:

```
daemon: charon
version: 5.6.2
sysname: Linux
release: 4.15.0-72-generic
machine: x86_64
```

## Streamed Command Requests

Another important concept in vici is server-issued events. A complete list of defined events can be found in the **Server-issued events** section of the [vici README](https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html). Some commands, for example the `list-certs` command, work by streaming a certain event type for the duration of the command request. In the case of `list-certs`, charon streams messages of the `list-cert` event type. We can make these types of commmand requests with `Session.StreamedCommandRequest`. When making a streamed command request, it is our job to tell the daemon which type of event we want to listen for during the command.

Let's continue with the `list-certs` example. We know the name of the command, `list-certs`, and we know we need to tell the daemon to stream `list-cert` events. If we look at the command's message parameters, we see some optional parameters:

```
{
    type = <certificate type to filter for, X509|X509_AC|X509_CRL|
                                            OCSP_RESPONSE|PUBKEY  or ANY>
    flag = <X.509 certificate flag to filter for, NONE|CA|AA|OCSP or ANY>
    subject = <set to list only certificates having subject>
} => {
    # completes after streaming list-cert events
}
```

Where each `list-cert` event contains the following information:

```
{
    type = <certificate type, X509|X509_AC|X509_CRL|OCSP_RESPONSE|PUBKEY>
    flag = <X.509 certificate flag, NONE|CA|AA|OCSP>
    has_privkey = <set if a private key for the certificate is available>
    data = <ASN1 encoded certificate data>
    subject = <subject string if defined and certificate type is PUBKEY>
    not-before = <time string if defined and certificate type is PUBKEY>
    not-after  = <time string if defined and certificate type is PUBKEY>
}
```

So, let's say we wanted to filter the results to only list CA certs. We can accomplish this by doing the following:

```go
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

        m := vici.NewMessage()
        
        if err := m.Set("flag", "CA"); err != nil {
                fmt.Println(err)
                return
        }

        ms, err := session.StreamedCommandRequest("list-certs", "list-cert", m)
        if err != nil {
                fmt.Println(err)
                return
        }

        for _, m := range ms.Messages() {
                if m.Err() != nil {
                        fmt.Println(err)
                        return
                }
                
                // Process CA cert information
                // ...        
        }
}
``` 

## Event Listener

A `Session` can also be used to listen for specific server-issued events at any time, not only during streamed command requests. This is done with the `Session.Subscribe` function, which accepts a list of event types. As an example, say we wanted to create a routine to monitor the state of a given SA, as well as `log` events. We can register the `Session`'s event listener to listen for the `ike-updown` and `log` events like this:

```go
package main

import (
	"context"
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

	// Subscribe to 'ike-updown' and 'log' events.
	if err := session.Subscribe("ike-updown", "log"); err != nil {
		fmt.Println(err)
		return
	}

	// IKE SA configuration name
	name := "rw"

	for {
		e, err := session.NextEvent(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}

                // The Event.Name field corresponds to the event name
                // we used to make the subscription. The Event.Message
                // field contains the Message from the server.
                switch e.Name{
                case "ike-updown":
		        state := e.Message.Get(name).(*vici.Message).Get("state")
		        fmt.Printf("IKE SA state changed (name=%s): %s\n", name, state)
                case "log":
                        // Log events contain a 'msg' field with the log message
                        fmt.Println(e.Message.Get("msg"))
                }
	}
}
```

The `Session.NextEvent` function is used to read messages from the listener, and will block until the listener has received an event from the server, or until the supplied context is cancelled. Event subscriptions and unsubscriptions can be made at any time while the `Session` is active.

## Message Marshaling

Some commands require a lot of parameters, or even a whole IKE SA configuration in the case of `load-conn`. Using `Message.Set` for this sort of thing is not very flexible and is quite cumbersome. The `MarshalMessage` function provides a way to easily construct a `Message` from a Go struct. To start with a simple example, let's define a struct `cert` that can be used to load certificates into the daemon using the `load-cert` command. If we look at the [vici README](https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html) again, we see the `load-cert` command's message parameters:

```
{
    type = <certificate type, X509|X509_AC|X509_CRL>
    flag = <X.509 certificate flag, NONE|CA|AA|OCSP>
    data = <PEM or DER encoded certificate data>
} => {
    success = <yes or no>
    errmsg = <error string on failure>
}
```

So our Go struct is simple:

```go
type cert struct {
        Type string `vici:"type"`
        Flag string `vici:"flag"`
        Data string `vici:"data"`
}
```

Remember, as stated on [godoc](https://godoc.org/github.com/strongswan/govici/vici#MarshalMessage), struct fields are only marshaled when they are exported and have a `vici` struct tag. Notice that the struct tags are identical to the field names in the `load-cert` message parameters. Now, we could wrap this all up into a helper function that loads a certificate into the daemon given its path on the filesystem.

```go
package main

import (
	"encoding/pem"
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
```

Pointer types can be useful to preserve defaults as specified in [swanctl.conf](https://wiki.strongswan.org/projects/strongswan/wiki/Swanctlconf) when those defaults do not align with Go zero values. For example, mobike is [enabled by default for IKEv2 connections](https://wiki.strongswan.org/projects/strongswan/wiki/Swanctlconf#connections-section), but if you have a `Mobike bool` field in your struct, the Go zero value will override the default behavior. In these situations, using `*bool` will result in the zero-value being `nil`, and the field will not be marshaled.

## Putting it all together

For a more complicated example, let's use `load-conn` to load an IKE SA configuration. The real work in doing this is defining some types to represent our configuration. The [swanctl.conf](https://wiki.strongswan.org/projects/strongswan/wiki/Swanctlconf) documentation is the best place to look for the information we need about configuration options and structure. For our case, let's take a `swanctl.conf` from the testing environment:

```
connections {

   rw {
      local_addrs  = 192.168.0.1

      local {
         auth = pubkey
         certs = moonCert.pem
         id = moon.strongswan.org
      }
      remote {
         auth = pubkey
      }
      children {
         net {
            local_ts  = 10.1.0.0/16 

            updown = /usr/local/libexec/ipsec/_updown iptables
            esp_proposals = aes128gcm128-x25519
         }
      }
      version = 2
      proposals = aes128-sha256-x25519
   }
}
```

We'll create Go types that satisfy the needs of this specific configuration, a common "road warrior" scenario with certificate-based authentication. See [here](https://www.strongswan.org/testing/testresults/swanctl/rw-cert/) for details on this testing scenario. If you're more familiar with the `ipsec.conf` configuration format, see [this document](https://wiki.strongswan.org/projects/strongswan/wiki/Fromipsecconf) for help migrating to the `swanctl.conf` format.

We can start by defining a type for a connection, where the fields correspond to the [`connections.<conn>.*`](https://wiki.strongswan.org/projects/strongswan/wiki/Swanctlconf#connections-section) fields defined in the `swanctl.conf` documentation.

```go
type connection struct {
	Name string // This field will NOT be marshaled!

	LocalAddrs []string            `vici:"local_addrs"`
	Local      *localOpts          `vici:"local"`
	Remote     *remoteOpts         `vici:"remote"`
	Children   map[string]*childSA `vici:"children"`
	Version    int                 `vici:"version"`
	Proposals  []string            `vici:"proposals"`
}
```

Then, we need to define `localOpts` and `remoteOpts` as referenced in the above definition:

```go
type localsOpts struct {
	Auth  string   `vici:"auth"`
	Certs []string `vici:"certs"`
	ID    string   `vici:"id"`
}

type remoteOpts struct {
	Auth string `vici:"auth"`
}
```

Remember, in this example, we only include the fields that are needed for our particular swanctl.conf. But any options from the [`connections.<conn>.local<suffix>`](https://wiki.strongswan.org/projects/strongswan/wiki/Swanctlconf#connectionsltconngtlocalltsuffixgt-section) or [`connections.<conn>.remote<suffix>`](https://wiki.strongswan.org/projects/strongswan/wiki/Swanctlconf#connectionsltconngtremoteltsuffixgt-section) sections could be defined here.

Finally, we need a `childSA` type:

```go
type childSA struct {
	LocalTrafficSelectors []string `vici:"local_ts"`
	Updown                string   `vici:"updown"`
	ESPProposals          []string `vici:"esp_proposals"`
}
```

Putting this all together, we can write some helpers to load our configuration into the daemon, and then establish the SAs.

```go
package main

import (
        "github.com/strongswan/govici/vici"
)

type connection struct {
	Name string // This field will NOT be marshaled!

	LocalAddrs []string            `vici:"local_addrs"`
	Local      *localOpts          `vici:"local"`
	Remote     *remoteOpts         `vici:"remote"`
	Children   map[string]*childSA `vici:"children"`
	Version    int                 `vici:"version"`
	Proposals  []string            `vici:"proposals"`
}

type localOpts struct {
	Auth  string   `vici:"auth"`
	Certs []string `vici:"certs"`
	ID    string   `vici:"id"`
}

type remoteOpts struct {
	Auth string `vici:"auth"`
}

type childSA struct {
	LocalTrafficSelectors []string `vici:"local_ts"`
	Updown                string   `vici:"updown"`
	ESPProposals          []string `vici:"esp_proposals"`
}

func loadConn(conn connection) error {
	s, err := vici.NewSession()
	if err != nil {
		return err
	}
        defer s.Close()

	c, err := vici.MarshalMessage(&conn)
	if err != nil {
		return err
	}

        m := vici.NewMessage()
        if err := m.Set(conn.Name, c); err != nil {
                return err
        }

	_, err = s.CommandRequest("load-conn", m)

	return err
}

func initiate(ike, child string) error {
	s, err := vici.NewSession()
	if err != nil {
		return err
	}
        defer s.Close()

	m := vici.NewMessage()

	if err := m.Set("child", child); err != nil {
		return err
	}

	if err := m.Set("ike", ike); err != nil {
		return err
	}

	ms, err := s.StreamedCommandRequest("initiate", "control-log", m)
	if err != nil {
		return err
	}

	for _, msg := range ms.Messages() {
		if err := msg.Err(); err != nil {
                        return err
		}
	}

	return nil
}
```
