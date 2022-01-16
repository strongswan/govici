# Changelog
All notable changes to govici will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

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

[Unreleased]: https://github.com/strongswan/govici/compare/v0.5.2...HEAD
[v0.4.0]: https://github.com/strongswan/govici/compare/v0.3.0...v0.4.0
[v0.4.1]: https://github.com/strongswan/govici/compare/v0.4.0...v0.4.1
[v0.5.0]: https://github.com/strongswan/govici/compare/v0.4.1...v0.5.0
[v0.5.1]: https://github.com/strongswan/govici/compare/v0.5.0...v0.5.1
[v0.5.2]: https://github.com/strongswan/govici/compare/v0.5.1...v0.5.2
