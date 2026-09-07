.PHONY: deps run test fmt build

GO_CMD := GO111MODULE=on go

deps:
	$(GO_CMD) mod tidy

run:
	$(GO_CMD) run .

test:
	$(GO_CMD) test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

build:
	mkdir -p bin
	$(GO_CMD) build -o bin/friends-records .
