# govici

[![Lint](https://github.com/strongswan/govici/actions/workflows/lint.yaml/badge.svg)](https://github.com/strongswan/govici/actions/workflows/lint.yaml?query=branch%3Amaster)
[![Tests](https://github.com/strongswan/govici/actions/workflows/test.yaml/badge.svg)](https://github.com/strongswan/govici/actions/workflows/test.yaml?query=branch%3Amaster)
[![Go Reference](https://pkg.go.dev/badge/github.com/strongswan/govici/vici.svg)](https://pkg.go.dev/github.com/strongswan/govici/vici)

## About

The strongSwan [vici protocol](https://docs.strongswan.org/docs/latest/plugins/vici.html) is used for external applications to monitor, configure, and control the IKE daemon charon. This Go package provides a pure-go implementation of a vici client library.

The package documentation can be found on [pkg.go.dev](https://pkg.go.dev/github.com/strongswan/govici/vici).

### API Stability

This package makes an effort to not make breaking changes to the API, but while it is in early stages it may be necessary. The goal is to be able to guarantee API stability at `v1.0.0`. For details on changes to the API, please read the [changelog](CHANGELOG.md).

When a new minor version is released, the previous minor version will still receive updates for bug fixes if needed, especially when the new minor version introduces breaking changes.

## Getting started

```go
import (
        "github.com/strongswan/govici/vici"
)
```

This package does not implement wrappers for individual vici commands, nor does it pre-define types for the message parameter of those commands. Commands are made by passing a command name and a populated `Message` to the `Session.Call` function. For a detailed walkthrough on how to use this package, see [Getting Started with vici](docs/getting_started.md).

There are additional examples for some functions on [pkg.go.dev](https://pkg.go.dev/github.com/strongswan/govici/vici).
