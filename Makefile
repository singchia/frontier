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
	make clean -C examples
	make clean -C test/bench

.PHONY: install
install:
	install -m 0755 -d $(DESTDIR)$(BINDIR)
	install -m 0755 -d $(DESTDIR)$(CONFDIR)
	install -m 0755 ./frontier $(DESTDIR)$(BINDIR)
	install -m 0755 ./pkg/config/config.yaml $(DESTDIR)$(CONFDIR)

.PHONY: image
image:
	docker buildx build -t frontier:${VERSION} .

.PHONY: container
container:
	docker rm -f frontier
	docker run -d --name frontier -p 2431:2431 -p 2432:2432 frontier:${VERSION} 

.PHONY: bench
bench: container
	make bench -C test/bench

.PHONY: output
output: build


