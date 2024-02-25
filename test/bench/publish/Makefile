GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: publish_edge publish_service

.PHONY: clean
clean:
	rm publish_edge publish_service

publish_edge: edge/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o publish_edge edge/*.go

publish_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o publish_service service/*.go

bench: publish_edge publish_service
	chmod +x ./publish.sh && ./publish.sh