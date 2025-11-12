# Changelog
All notable changes to govici will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [v0.8.0] - 2025-11-11

### Added

- Session.Call, a context-aware alternative to Session.CommandRequest.
- Session.CallStreaming, a context-aware alternative to Session.StreamedCommandRequest. Another important distinction is that Session.CallStreaming returns an `iter.Seq2[*vici.Message, error]` so that the call does not block until all events associated with the call have been received (as is the case with Session.StreamedCommandRequest). Instead, a caller can iterate over the returned `iter.Seq2[*vici.Message, error]` to handle the events as they come in.
- Message.String() implementation. This provides a convenient way to get a string representation of a Message.

### Changed

- Require go 1.24.
- Re-factor transport into clientConn type that is context-aware.

### Deprecated

- Deprecated Session.CommandRequest in favor of Session.Call. This will be removed in a future release.
- Deprecated Session.StreamedCommandRequest in favor of Session.CallStreaming. This will be removed in a future release.

## [v0.7.0] - 2023-02-24

### Changed
- Fix a bug where mutex may not be Unlock()'d in Session.Close().
- Make sure registered event channels are closed when the event listener exits:
  https://github.com/strongswan/govici/issues/46.
- Prevent panics after closing the session.
- Set GOOS appropriate defaults at build time.
- Bring the CI up to date and fix some linting errors.
- Lazily start the event listener loop the first time Subscribe() is called.

### Removed
- Session.NextEvent API.
- MessageStream type.

## [v0.6.0] - 2022-01-19

### Added
- Session.NotifyEvents API.
- Session.StopEvents API.
- NewMessageStream function.
- More package-level documentation for pkg.go.dev page.

### Changed
- NextEvent will not block if it receives an event and the event channel buffer is full.
- NextEvent is deprecated in favor of NotifyEvents, and will be removed prior to v1.0.
- MessageStream type is deprecated, and will be removed prior to v1.0.
- Fix an error message related to trying to unmarshal into the wrong type.

## [v0.5.2] - 2021-08-24

### Changed
- Fixed https://github.com/strongswan/govici/issues/34.
- Simplified event error handling code, and event listener control flow.

## [v0.5.1] - 2021-04-12

### Added
- GitHub workflows.

### Changed
- Event listener does not send unnecessary errors to event channel.
- Simplified some internal functions, like packet.isNamed() and Message.elements().

## [v0.5.0] - 2020-09-14

### Added
- New `inline` tag option for inlining embedded structs.
- Explicitly define "empty" message element so that it is clear when a field
  will not be marshaled into a Message.

## [v0.4.1] - 2020-08-10

### Changed
- Behavior of handling io.EOF error in event listener to avoid potential deadlock.

## [v0.4.0] - 2020-07-03

### Added
- CHANGELOG.md particularly to help track API changes pre-v1.0.0.
- Session.Subscribe/Unsubscribe/UnsubscribeAll APIs for event subscription.
- WithAddr SessionOption to specify address that charon is listening on.
- WithDialContext SessionOption to provide a dial func to Session.
- Expose Event type with exported Name and Message fields.
- Add a timestamp to Event type.

### Changed
- Behavior of event registration. Package users can now subscribe/unsubscribe at
  any time while the Session is active.
- Event listener is always active, and the same listen() loop now services event
  registration responses as well as event packets.
- NextEvent accepts a context.Context so that it can be cancelled by the caller.

### Removed
- Session.Listen API.

[Unreleased]: https://github.com/strongswan/govici/compare/v0.8.0...HEAD
[v0.4.0]: https://github.com/strongswan/govici/compare/v0.3.0...v0.4.0
[v0.4.1]: https://github.com/strongswan/govici/compare/v0.4.0...v0.4.1
[v0.5.0]: https://github.com/strongswan/govici/compare/v0.4.1...v0.5.0
[v0.5.1]: https://github.com/strongswan/govici/compare/v0.5.0...v0.5.1
[v0.5.2]: https://github.com/strongswan/govici/compare/v0.5.1...v0.5.2
[v0.6.0]: https://github.com/strongswan/govici/compare/v0.5.2...v0.6.0
[v0.7.0]: https://github.com/strongswan/govici/compare/v0.6.0...v0.7.0
[v0.8.0]: https://github.com/strongswan/govici/compare/v0.7.0...v0.8.0
