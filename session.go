//
// Copyright (C) 2019 Nick Rosbrook
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

// Package vici implements a strongSwan vici protocol client
package vici

import (
	"sync"
)

// Session is a vici client session
type Session struct {
	// Only one command can be active on the transport at a time,
	// but events may get raised at any time while registered, even
	// during an active command request command. So, give session two
	// transports: one is locked with mutex during use, e.g. command
	// requests (including streamed requests), and the other is used
	// for listening to registered events.
	mux sync.Mutex
	ctr *transport

	el *eventListener
}

// NewSession returns a new vici session.
func NewSession() (*Session, error) {
	ctr, err := newTransport(nil)
	if err != nil {
		return nil, err
	}
	elt, err := newTransport(nil)
	if err != nil {
		return nil, err
	}

	s := &Session{
		ctr: ctr,
		el:  newEventListener(elt),
	}

	return s, nil
}

// Version returns daemon and system specific version information.
func (s *Session) Version() (*Message, error) {
	return s.sendRequest("version", nil)
}

// Stats returns IKE daemon statistics and load information.
func (s *Session) Stats() (*Message, error) {
	return s.sendRequest("stats", nil)
}

// ReloadSettings reloads strongswan.conf settings and all plugins supporting
// configuration reload.
func (s *Session) ReloadSettings() (*Message, error) {
	return s.sendRequest("reload-settings", nil)
}

// Initiate initiates an SA.
func (s *Session) Initiate(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("initiate", ControlLog, msg)
}

// Terminate terminates an SA.
func (s *Session) Terminate(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("terminate", ControlLog, msg)
}

// Rekey initiates the re-keying of an SA.
func (s *Session) Rekey(msg *Message) (*Message, error) {
	return s.sendRequest("rekey", msg)
}

// Redirect redirects a client-initiated IKE_SA to another gateway, only for IKEv2 and
// if supported by the peer.
func (s *Session) Redirect(msg *Message) (*Message, error) {
	return s.sendRequest("redirect", msg)
}

// Install installs a trap, drop or bypass policy defined by a CHILD_SA config.
func (s *Session) Install(msg *Message) (*Message, error) {
	return s.sendRequest("install", msg)
}

// Uninstall uninstalls a trap, drop or bypass policy defined by a CHILD_SA config.
func (s *Session) Uninstall(msg *Message) (*Message, error) {
	return s.sendRequest("uninstall", msg)
}

// ListSAs lists currently active IKE_SAs and associated CHILD_SAs by streaming `list-sa`
// events.
func (s *Session) ListSAs(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("list-sas", ListSA, msg)
}

// ListPolicies lists currently active trap, drop and bypass policies by streaming
// `list-policy` events.
func (s *Session) ListPolicies(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("list-policies", ListPolicy, msg)
}

// ListConns lists currently loaded connections by streaming `list-conn` events, which includes
// all connections known by the daemon, not only those loaded over vici.
func (s *Session) ListConns(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("list-conns", ListConn, msg)
}

// GetConns returns a list of connection names exclusively loaded over vici, not including connections
// found in other backends.
func (s *Session) GetConns() (*Message, error) {
	return s.sendRequest("get-conns", nil)
}

// ListCerts lists currently loaded certificates by streaming `list-cert` events, which includes all
// certificates known by the daemon, not only those loaded over vici.
func (s *Session) ListCerts(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("lsit-certs", ListCert, msg)
}

// ListAuthorities lists currently loaded CA information by streaming `list-authority` events.
func (s *Session) ListAuthorities(msg *Message) (*MessageStream, error) {
	return s.sendStreamedRequest("list-authorities", ListAuthority, msg)
}

// GetAuthorities returns a list of currently loaded CA names.
func (s *Session) GetAuthorities() (*Message, error) {
	return s.sendRequest("get-authorities", nil)
}

// LoadConn loads a single connection definition to the daemon. An existing connection with the same name
// gets updated or replaced.
func (s *Session) LoadConn(msg *Message) (*Message, error) {
	return s.sendRequest("load-conn", msg)
}

// UnloadConn unloads a previously loaded connection by name.
func (s *Session) UnloadConn(msg *Message) (*Message, error) {
	return s.sendRequest("unload-conn", msg)
}

// LoadCert loads a certificate into the daemon.
func (s *Session) LoadCert(msg *Message) (*Message, error) {
	return s.sendRequest("load-cert", msg)
}

// LoadKey loads a private key into the daemon.
func (s *Session) LoadKey(msg *Message) (*Message, error) {
	return s.sendRequest("load-key", msg)
}

// UnloadKey unloads a key with the given key identifier.
func (s *Session) UnloadKey(msg *Message) (*Message, error) {
	return s.sendRequest("unload-key", msg)
}

// GetKeys returns a list of identifiers of private keys loaded exclusively over vici, not including keys
// found in other backends.
func (s *Session) GetKeys() (*Message, error) {
	return s.sendRequest("get-keys", nil)
}

// LoadToken loads a private key located on a token into the daemon. Such keys may be listed and unloaded using the
// get-keys and unload-key commands, respectively (based on the key identifier derived from the public key).
func (s *Session) LoadToken(msg *Message) (*Message, error) {
	return s.sendRequest("load-token", msg)
}

// LoadShared loads a shared IKE PSK, EAP, XAuth or NTLM secret into the daemon.
func (s *Session) LoadShared(msg *Message) (*Message, error) {
	return s.sendRequest("load-shared", msg)
}

// UnloadShared unloads a previously shared IKE PSK, EAP, XAuth or NTLM secret by its unique identifier.
func (s *Session) UnloadShared(msg *Message) (*Message, error) {
	return s.sendRequest("unload-shared", msg)
}

// GetShared returns a list of unique identifiers of shared keys loaded exclusively over vici, not including
// keys found in other backends.
func (s *Session) GetShared() (*Message, error) {
	return s.sendRequest("get-shared", nil)
}

// FlushCerts flushes the certificate cache. The optional type argument allows to flush only certificates of
// a given type, e.g. cached CRLs.
func (s *Session) FlushCerts(msg *Message) (*Message, error) {
	return s.sendRequest("flush-certs", msg)
}

// CleadCreds clears all loaded certificate, private key and shared key credentials. This only affects credentials
// loaded over vici but additionally flushes the credential cache.
func (s *Session) ClearCreds() (*Message, error) {
	return s.sendRequest("clear-creds", nil)
}

// LoadAuthority loads a single CA definition into the daemon. An exisiting authority with the same name gets replaced.
func (s *Session) LoadAuthority(msg *Message) (*Message, error) {
	return s.sendRequest("load-authority", msg)
}

// UnloadAuthority unloads a previously loaded CA definition by name.
func (s *Session) UnloadAuthority(msg *Message) (*Message, error) {
	return s.sendRequest("unload-authority", msg)
}

// LoadPool loads an in-memory virtual IP and configuration attribute pool. Exisiting pools with the same name
// get updated, if possible.
func (s *Session) LoadPool(msg *Message) (*Message, error) {
	return s.sendRequest("load-pool", msg)
}

// UnloadPool unloads a previously loaded virtual IP and configuration attribute pool. Unloading fails for pools
// with leases currently online.
func (s *Session) UnloadPool(msg *Message) (*Message, error) {
	return s.sendRequest("unload-pool", msg)
}

// GetPools lists the currently loaded pools.
func (s *Session) GetPools(msg *Message) (*Message, error) {
	return s.sendRequest("get-pools", msg)
}

// GetAlgorithms lists currently loaded algorithms and their implementation.
func (s *Session) GetAlgorithms() (*Message, error) {
	return s.sendRequest("get-algorithms", nil)
}

// GetCounters lists global or connection-specific counters for several IKE events.
func (s *Session) GetCounters(msg *Message) (*Message, error) {
	return s.sendRequest("get-counters", msg)
}

// ResetCounters resets global or connection-specific IKE event counters.
func (s *Session) ResetCounters(msg *Message) (*Message, error) {
	return s.sendRequest("reset-counters", msg)
}

// Listen listens for registered events.
func (s *Session) Listen(events []string) error {
	return s.el.safeListen(events)
}

// NextEvent returns the next event seen by Listen. NextEvent is a blocking call - if there
// is no event in the event buffer, NextEvent will wait until there is.
func (s *Session) NextEvent() (*Message, error) {
	return s.el.nextEvent()
}
