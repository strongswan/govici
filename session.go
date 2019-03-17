// Package vici implements a strongSwan vici protocol client
package vici

// Session is a vici client session
type Session struct{}

// Version returns daemon and system specific version information.
func (s *Session) Version() {}

// Stats returns IKE daemon statistics and load information.
func (s *Session) Stats() {}

// ReloadSettings reloads strongswan.conf settings and all plugins supporting
// configuration reload.
func (s *Session) ReloadSettings() {}

// Initiate initiates an SA.
func (s *Session) Initiate() {}

// Terminate terminates an SA.
func (s *Session) Terminate() {}

// Rekey initiates the re-keying of an SA.
func (s *Session) Rekey() {}

// Redirect redirects a client-initiated IKE_SA to another gateway, only for IKEv2 and
// if supported by the peer.
func (s *Session) Redirect() {}

// Install installs a trap, drop or bypass policy defined by a CHILD_SA config.
func (s *Session) Install() {}

// Uninstall uninstalls a trap, drop or bypass policy defined by a CHILD_SA config.
func (s *Session) Uninstall() {}

// ListSAs lists currently active IKE_SAs and associated CHILD_SAs by streaming `list-sa`
// events.
func (s *Session) ListSAs() {}

// ListPolicies lists currently active trap, drop and bypass policies by streaming
// `list-policy` events.
func (s *Session) ListPolicies() {}

// ListConns lists currently loaded connections by streaming `list-conn` events, which includes
// all connections known by the daemon, not only those loaded over vici.
func (s *Session) ListConns() {}

// GetConns returns a list of connection names exclusively loaded over vici, not including connections
// found in other backends.
func (s *Session) GetConns() {}

// ListCerts lists currently loaded certificates by streaming `list-cert` events, which includes all
// certificates known by the daemon, not only those loaded over vici.
func (s *Session) ListCerts() {}

// ListAuthorities lists currently loaded CA information by streaming `list-authority` events.
func (s *Session) ListAuthorities() {}

// GetAuthorities returns a list of currently loaded CA names.
func (s *Session) GetAuthorities() {}

// LoadConn loads a single connection definition to the daemon. An existing connection with the same name
// gets updated or replaced.
func (s *Session) LoadConn() {}

// UnloadConn unloads a previously loaded connection by name.
func (s *Session) UnloadConn() {}

// LoadCert loads a certificate into the daemon.
func (s *Session) LoadCert() {}

// LoadKey loads a private key into the daemon.
func (s *Session) LoadKey() {}

// UnloadKey unloads a key with the given key identifier.
func (s *Session) UnloadKey() {}

// GetKeys returns a list of identifiers of private keys loaded exclusively over vici, not including keys
// found in other backends.
func (s *Session) GetKey() {}

// LoadToken loads a private key located on a token into the daemon. Such keys may be listed and unloaded using the
// get-keys and unload-key commands, respectively (based on the key identifier derived from the public key).
func (s *Session) LoadToken() {}

// LoadShared loads a shared IKE PSK, EAP, XAuth or NTLM secret into the daemon.
func (s *Session) LoadShared() {}

// UnloadShared unloads a previously shared IKE PSK, EAP, XAuth or NTLM secret by its unique identifier.
func (s *Session) UnloadShared() {}

// GetShared returns a list of unique identifiers of shared keys loaded exclusively over vici, not including
// keys found in other backends.
func (s *Session) GetShared() {}

// FlushCerts flushes the certificate cache. The optional type argument allows to flush only certificates of
// a given type, e.g. cached CRLs.
func (s *Session) FlushCerts() {}

// CleadCreds clears all loaded certificate, private key and shared key credentials. This only affects credentials
// loaded over vici but additionally flushes the credential cache.
func (s *Session) ClearCreds() {}

// LoadAuthority loads a single CA definition into the daemon. An exisiting authority with the same name gets replaced.
func (s *Session) LoadAuthority() {}

// UnloadAuthority unloads a previously loaded CA definition by name.
func (s *Session) UnloadAuthority() {}

// LoadPool loads an in-memory virtual IP and configuration attribute pool. Exisiting pools with the same name
// get updated, if possible.
func (s *Session) LoadPool() {}

// UnloadPool unloads a previously loaded virtual IP and configuration attribute pool. Unloading fails for pools
// with leases currently online.
func (s *Session) UnloadPool() {}

// GetPools lists the currently loaded pools.
func (s *Session) GetPools() {}

// GetAlgorithms lists currently loaded algorithms and their implementation.
func (s *Session) GetAlgoritms() {}

// GetCounters lists global or connection-specific counters for several IKE events.
func (s *Session) GetCounters() {}

// ResetCounters resets global or connection-specific IKE event counters.
func (s *Session) ResetCounters() {}
