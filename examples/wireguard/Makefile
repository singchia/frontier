PREFIX?=/usr
BINDIR?=$(PREFIX)/bin

GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: wg-edge wg-router udpping

.PHONY: clean
clean:
	rm -f ./bin/wg-edge ./bin/wg-router ./bin/udpping

.PHONY: wg-edge
wg-edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/wg-edge ./edge

.PHONY: wg-router
wg-router: router/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/wg-router ./router

.PHONY: udpping
udpping: cmd/udpping/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/udpping ./cmd/udpping
