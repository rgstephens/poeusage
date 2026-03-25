APP     := poeusage
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/gstephens/poeusage/cmd.Version=$(VERSION) \
	-X github.com/gstephens/poeusage/cmd.BuildDate=$(DATE)

.PHONY: build run test tidy clean snapshot release

## Build the binary for the current platform
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APP) .

## Run the tool directly (pass ARGS="..." to supply arguments)
run:
	go run -ldflags "$(LDFLAGS)" . $(ARGS)

## Run tests
test:
	go test ./...

## Tidy dependencies
tidy:
	go mod tidy

## Clean build output
clean:
	rm -rf bin/ dist/

## Build a local GoReleaser snapshot (all platforms, no publish)
snapshot:
	goreleaser release --snapshot --clean

## Tag and trigger a full release via GitHub Actions (set TAG=vX.Y.Z)
release:
	@if [ -z "$(TAG)" ]; then echo "Usage: make release TAG=v1.2.3"; exit 1; fi
	git tag $(TAG)
	git push origin $(TAG)
