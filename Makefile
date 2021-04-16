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
	golangci-lint --verbose run

.PHONY: docs
docs: docs-code-examples
	embedmd -w _docs/*.md

.PHONY: docs-code-examples
docs-code-examples: _docs/code/*.go
	for file in $^ ; do \
	    go build -o /dev/null $${file} ; \
	done
