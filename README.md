# govici

[![Lint](https://github.com/strongswan/govici/actions/workflows/lint.yaml/badge.svg)](https://github.com/strongswan/govici/actions/workflows/lint.yaml?query=branch%3Amaster)
[![Tests](https://github.com/strongswan/govici/actions/workflows/test.yaml/badge.svg)](https://github.com/strongswan/govici/actions/workflows/test.yaml?query=branch%3Amaster)
[![Go Reference](https://pkg.go.dev/badge/github.com/strongswan/govici/vici.svg)](https://pkg.go.dev/github.com/strongswan/govici/vici)
[![Go Report Card](https://goreportcard.com/badge/github.com/strongswan/govici/vici)](https://goreportcard.com/report/github.com/strongswan/govici/vici)

## About

The strongSwan [vici protocol](https://www.strongswan.org/apidoc/md_src_libcharon_plugins_vici_README.html) is used for external applications to monitor, configure, and control the IKE daemon charon. This Go package provides a pure-go implementation of a vici client library.

The package documentation can be found on [godoc](https://godoc.org/github.com/strongswan/govici/vici).

### API Stability

This package makes an effort to not make breaking changes to the API, but while it is in early stages it may be necessary. The goal is to be able to guarantee API stability at `v1.0.0`. For details on changes to the API, please read the [changelog](CHANGELOG.md).

## Getting started

```go
import (
        "github.com/strongswan/govici/vici"
)
```

This package does not implement wrappers for individual vici commands, nor does it pre-define types for the message parameter of those commands. Commands are made by passing a command name and a populated `Message` to the `Session.CommandRequest` function. For a detailed walkthrough on how to use this package, see [Getting Started with vici](docs/getting_started.md).

There are additional examples for some functions on [godoc](https://godoc.org/github.com/strongswan/govici/vici).
