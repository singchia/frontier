GOHOSTOS?=$(shell go env GOHOSTOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all
all: iclm

.PHONY: iclm
iclm:
	make -C iclm

.PHONY: clean
clean:
	make clean -C iclm