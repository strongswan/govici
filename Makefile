GOPATH ?= $(HOME)/go
PATH := $(GOPATH)/bin:$(PATH)

.PHONY: check
check: golint test

.PHONY: test
test:
	go test -v ./

.PHONY: golint
golint:
	golangci-lint --verbose run --enable-all -Dgochecknoglobals -Dgochecknoinits -Dlll
