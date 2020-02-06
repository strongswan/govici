GOPATH ?= $(HOME)/go
PATH := $(GOPATH)/bin:$(PATH)

TEST_FLAGS ?=

.PHONY: check
check: golint test

.PHONY: test
test:
	go test -v ./vici -count=1 $(TEST_FLAGS)

.PHONY: golint
golint:
	golangci-lint --verbose run --enable-all -Dgochecknoglobals -Dgochecknoinits -Dlll --exclude unused
