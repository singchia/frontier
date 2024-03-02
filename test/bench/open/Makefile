GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: open_edge open_service

.PHONY: clean
clean:
	rm open_edge open_service || true

open_edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o open_edge edge/*.go

open_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o open_service service/*.go

bench: open_edge open_service
	chmod +x ./open.sh && ./open.sh