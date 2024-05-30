PREFIX?=/usr
BINDIR?=$(PREFIX)/bin

GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: chatroom_service chatroom_client

.PHONY: clean
clean:
	rm chatroom_service chatroom_client

.PHONY: chatroom_service
chatroom_service: service/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/chatroom_service service/*.go

.PHONY: chatroom_client
chatroom_client: client/*.go
	CGO_ENABLED=0 GOOS=$(GOHOSTOS) GOARCH=$(GOARCH) \
		go build -trimpath -ldflags "-s -w" -o ./bin/chatroom_client client/*.go