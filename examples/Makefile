GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: iclm chatroom

.PHONY: iclm
iclm:
	make -C iclm

.PHONY: chatroom
chatroom:
	make -C chatroom

.PHONY: clean
clean:
	make clean -C iclm
	make clean -C chatroom