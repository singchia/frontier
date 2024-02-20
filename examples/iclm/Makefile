GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: iclm_service iclm_edge

.PHONY: clean
clean:
	rm iclm_service iclm_edge

iclm_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o iclm_service service/*.go

iclm_edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o iclm_edge edge/*.go