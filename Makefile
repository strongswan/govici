GOPATH ?= $(HOME)/go
PATH := $(GOPATH)/bin:$(PATH)

.PHONY: check
check: golint test

.PHONY: test
test:
	go test -v ./ -count=1

.PHONY: golint
golint:
	golangci-lint --verbose run --enable-all -Dgochecknoglobals -Dgochecknoinits -Dlll --exclude unused
