include ./Makefile.defs

CC?=cc

all: frontier examples

.PHONY: frontier
frontier:
	CC=${CC} CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o ./frontier cmd/frontier/main.go

.PHONY: frontier-linux
frontier-linux:
	CC=${CC} GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o ./frontier cmd/frontier/main.go

.PHONY: examples
examples:
	make -C examples

.PHONY: clean
clean:
	rm ./frontier || true
	make clean -C examples
	make clean -C test/bench

.PHONY: install
install:
	install -m 0755 -d $(DESTDIR)$(BINDIR)
	install -m 0755 -d $(DESTDIR)$(CONFDIR)
	install -m 0755 ./frontier $(DESTDIR)$(BINDIR)
	install -m 0755 ./pkg/frontier/config/frontier.yaml $(DESTDIR)$(CONFDIR)

.PHONY: image
image:
	docker buildx build -t frontier:${VERSION} -f images/Dockerfile.frontier .

.PHONY: image-gen-api
image-gen-api:
	docker buildx build -t image-gen-api:${VERSION} -f images/Dockerfile.controlplane-api .

.PHONY: image-gen-swagger
image-gen-swagger:
	docker buildx build -t frontier-gen-swagger:${VERSION} -f images/Dockerfile.controlplane-swagger .

.PHONY: container-frontier
container-frontier:
	docker rm -f frontier
	docker run -d --name frontier -p 2431:2431 -p 2432:2432 frontier:${VERSION} --config /usr/conf/frontier.yaml -v 5

.PHONY: frontier-api
frontier-api:
	docker run --rm -v ${PWD}/api/controlplane/frontier/v1:/api/controlplane/frontier/v1 image-gen-api:${VERSION}

.PHONY: frontlas-api
frontlas-api:
	docker run --rm -v ${PWD}/api/controlplane/frontlas/v1:/api/controlplane/frontlas/v1 image-gen-api:${VERSION}

.PHONY: bench
bench: container
	make bench -C test/bench

.PHONY: swagger
swagger:
	docker run --rm -v ${PWD}:/frontier frontier-gen-swagger:${VERSION}

.PHONY: output
output: build


