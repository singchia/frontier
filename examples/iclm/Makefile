PREFIX?=/usr
BINDIR?=$(PREFIX)/bin

GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: iclm_service iclm_edge

.PHONY: clean
clean:
	rm iclm_service iclm_edge

.PHONY: iclm_service
iclm_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/iclm_service service/*.go

.PHONY: iclm_edge
iclm_edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/iclm_edge edge/*.go