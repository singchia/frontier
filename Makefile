include ./Makefile.defs

REGISTRY?=singchia
CC?=cc

all: frontier frontlas

# binary
.PHONY: frontier
frontier:
	CC=${CC} CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o ./bin/frontier cmd/frontier/main.go

.PHONY: frontier-linux
frontier-linux:
	CC=${CC} GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o ./bin/frontier cmd/frontier/main.go

.PHONY: frontlas
frontlas:
	CC=${CC} CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o ./bin/frontlas cmd/frontlas/main.go

.PHONY: frontlas-linux
frontlas-linux:
	CC=${CC} GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o  ./bin/frontlas cmd/frontlas/main.go

# example
.PHONY: examples
examples:
	make -C examples
	mv examples/iclm/bin/* ./bin/ && rm -rf examples/iclm/bin
	mv examples/chatroom/bin/* ./bin/ && rm -rf examples/chatroom/bin

# clean
.PHONY: clean
clean:
	rm ./bin/* || true
	make clean -C examples
	make clean -C test/bench

# install
.PHONY: install-frontier
install-frontier: frontier
	install -m 0755 -d $(DESTDIR)$(BINDIR)
	install -m 0755 -d $(DESTDIR)$(CONFDIR)
	install -m 0755 ./bin/frontier $(DESTDIR)$(BINDIR)
	install -m 0755 ./etc/frontier.yaml $(DESTDIR)$(CONFDIR)

.PHONY: install-frontlas
install-frontlas: frontlas
	install -m 0755 -d $(DESTDIR)$(BINDIR)
	install -m 0755 -d $(DESTDIR)$(CONFDIR)
	install -m 0755 ./bin/frontlas $(DESTDIR)$(BINDIR)
	install -m 0755 ./etc/frontlas.yaml $(DESTDIR)$(CONFDIR)

.PHONY: install-example-iclm
install-example-iclm: examples
	install -m 0755 -d $(DESTDIR)$(BINDIR)
	install -m 0755 ./bin/iclm_service $(DESTDIR)$(BINDIR)
	install -m 0755 ./bin/iclm_edge $(DESTDIR)$(BINDIR)

# image
.PHONY: image-frontier
image-frontier:
	docker buildx build -t ${REGISTRY}/frontier:${VERSION} -f images/Dockerfile.frontier .

.PHONY: image-frontlas
image-frontlas:
	docker buildx build -t ${REGISTRY}/frontlas:${VERSION} -f images/Dockerfile.frontlas .

.PHONY: image-gen-api
image-gen-api:
	docker buildx build -t image-gen-api:${VERSION} -f images/Dockerfile.controlplane-api .

.PHONY: image-gen-swagger
image-gen-swagger:
	docker buildx build -t frontier-gen-swagger:${VERSION} -f images/Dockerfile.controlplane-swagger .

.PHONY: image-example-iclm
image-example-iclm:
	docker buildx build -t ${REGISTRY}/iclm_service:${VERSION} -f images/Dockerfile.example_iclm_service .

# push
.PHONY: push
push: push-frontier push-frontlas

.PHONY: push-frontier
push-frontier:
	docker push ${REGISTRY}/frontier:${VERSION}

.PHONY: push-frontlas
push-frontlas:
	docker push ${REGISTRY}/frontlas:${VERSION}

.PHONY: push-example-iclm
push-example-iclm:
	docker push ${REGISTRY}/iclm_service:${VERSION} 

# container
.PHONY: container
container: container-frontier container-frontlas

.PHONY: container-frontier
container-frontier:
	docker rm -f frontier
	docker run -d --name frontier -p 30011:30011 -p 30012:30012 ${REGISTRY}/frontier:${VERSION} --config /usr/conf/frontier.yaml -v 1

.PHONY: container-frontlas
container-frontlas:
	docker rm -f frontlas
	docker run -d --name frontlas -p 40011:40011 -p 40012:40012 ${REGISTRY}/frontlas:${VERSION} --config /usr/conf/frontlas.yaml -v 1

# api
.PHONY: api-frontier
api-frontier:
	docker run --rm -v ${PWD}/api/controlplane/frontier/v1:/api/controlplane/v1 image-gen-api:${VERSION}

.PHONY: api-frontlas
api-frontlas:
	docker run --rm -v ${PWD}/api/controlplane/frontlas/v1:/api/controlplane/v1 image-gen-api:${VERSION}

# bench
.PHONY: bench
bench: container-frontier
	make bench -C test/bench

.PHONY: swagger
swagger:
	docker run --rm -v ${PWD}:/frontier frontier-gen-swagger:${VERSION}

.PHONY: output
output: build


