GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: call_edge call_service

.PHONY: clean
clean:
	rm call_edge call_service || true

call_edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o call_edge edge/*.go

call_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o call_service service/*.go

bench: call_edge call_service
	chmod +x ./call.sh && ./call.sh