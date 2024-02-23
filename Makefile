include ./Makefile.defs

CC?=cc

all: frontier examples

.PHONY: frontier
frontier:
	CC=${CC} CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o ./frontier cmd/frontier/main.go

.PHONY: frontier-linux
frontier-linux:
	CC=${CC} GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w"-o ./frontier cmd/frontier/main.go

.PHONY: examples
examples:
	make -C examples

.PHONY: clean
clean:
	rm ./frontier
	rm ./examples/iclm/iclm_edge
	rm ./examples/iclm/iclm_service

.PHONY: install
install:
	install -m 0755 -d $(DESTDIR)$(BINDIR)
	install -m 0755 -d $(DESTDIR)$(CONFDIR)
	install -m 0755 ./frontier $(DESTDIR)$(BINDIR)
	install -m 0755 ./pkg/config/config.yaml $(DESTDIR)$(CONFDIR)

.PHONY: image
image:
	docker buildx build -t frontier:${VERSION} .

.PHONY: output
output: build


