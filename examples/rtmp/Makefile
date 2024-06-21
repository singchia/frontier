PREFIX?=/usr
BINDIR?=$(PREFIX)/bin

GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: rtmp_service rtmp_edge

.PHONY: clean
clean:
	rm rtmp_service rtmp_edge

.PHONY: rtmp_service
rtmp_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/rtmp_service service/*.go

.PHONY: rtmp_edge
rtmp_edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/rtmp_edge edge/*.go