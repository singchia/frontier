include ./Makefile.defs

REGISTRY?=singchia
CC?=cc

# Cross-compilation platforms
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64
PLATFORMS_LINUX := linux/amd64 linux/arm64
PLATFORMS_DARWIN := darwin/amd64 darwin/arm64
PLATFORMS_WINDOWS := windows/amd64 windows/arm64

# Output directory for cross-compiled binaries
DIST_DIR := dist/bin

# Default target
all: frontier frontlas

# Help target
.PHONY: help
help:
	@echo "Frontier Build System"
	@echo "===================="
	@echo ""
	@echo "Local builds (current platform):"
	@echo "  make frontier          - Build frontier for current platform"
	@echo "  make frontlas          - Build frontlas for current platform"
	@echo "  make all               - Build both frontier and frontlas"
	@echo ""
	@echo "Cross-compilation - All platforms:"
	@echo "  make build-all         - Build frontier and frontlas for all platforms"
	@echo "  make frontier-all      - Build frontier for all platforms"
	@echo "  make frontlas-all      - Build frontlas for all platforms"
	@echo ""
	@echo "Docker-based cross-compilation (recommended, supports CGO):"
	@echo "  make docker-build-all         - Build all using Docker (with CGO support)"
	@echo "  make docker-frontier-all     - Build frontier using Docker"
	@echo "  make docker-frontlas-all     - Build frontlas using Docker"
	@echo ""
	@echo "Cross-compilation - By OS:"
	@echo "  make frontier-linux-all    - Build frontier for Linux (amd64, arm64)"
	@echo "  make frontier-darwin-all   - Build frontier for macOS (amd64, arm64)"
	@echo "  make frontier-windows-all  - Build frontier for Windows (amd64, arm64)"
	@echo "  make frontlas-linux-all    - Build frontlas for Linux (amd64, arm64)"
	@echo "  make frontlas-darwin-all   - Build frontlas for macOS (amd64, arm64)"
	@echo "  make frontlas-windows-all - Build frontlas for Windows (amd64, arm64)"
	@echo ""
	@echo "Cross-compilation - Individual targets:"
	@echo "  make frontier-linux-amd64    - Build frontier for Linux amd64"
	@echo "  make frontier-linux-arm64    - Build frontier for Linux arm64"
	@echo "  make frontier-darwin-amd64   - Build frontier for macOS amd64"
	@echo "  make frontier-darwin-arm64   - Build frontier for macOS arm64"
	@echo "  make frontier-windows-amd64  - Build frontier for Windows amd64"
	@echo "  make frontier-windows-arm64 - Build frontier for Windows arm64"
	@echo ""
	@echo "Other targets:"
	@echo "  make clean            - Clean local build artifacts"
	@echo "  make clean-dist       - Clean cross-compilation artifacts"
	@echo "  make help             - Show this help message"
	@echo ""
	@echo "Output locations:"
	@echo "  Local builds:        ./bin/"
	@echo "  Cross-compilation:   $(DIST_DIR)/"
	@echo "  Binary naming:       {binary}-{os}-{arch} (e.g., frontier-linux-amd64)"
	@echo ""
	@echo "Note on CGO:"
	@echo "  - Native cross-compilation disables CGO (CGO_ENABLED=0)"
	@echo "  - This means SQLite backend won't work in cross-compiled binaries"
	@echo "  - Use 'buntdb' backend (default) for cross-compiled binaries"
	@echo "  - For CGO features, use 'make docker-build-all' (recommended)"

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

# Cross-compilation helpers
# Note: CGO is disabled for cross-compilation because:
# - Windows: CGO support is limited
# - Linux (from macOS): macOS C compiler cannot compile Linux-specific syscalls
# - macOS: CGO enabled only for native builds
# If you need CGO features (like SQLite), build natively on the target platform or use Docker
# Binary files are named as: {binary}-{os}-{arch} or {binary}-{os}-{arch}.exe (Windows)
define build-platform
	@echo "Building $(1)/$(2) for $(3)..."
	@mkdir -p $(DIST_DIR)
	@if [ "$(3)" = "windows" ]; then \
		GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(DIST_DIR)/$(4)-$(3)-$(2).exe $(5); \
	elif [ "$(3)" = "linux" ]; then \
		GOOS=$(1) GOARCH=$(2) CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o $(DIST_DIR)/$(4)-$(3)-$(2) $(5); \
	else \
		GOOS=$(1) GOARCH=$(2) CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o $(DIST_DIR)/$(4)-$(3)-$(2) $(5); \
	fi
endef

# Cross-compilation targets for frontier
.PHONY: frontier-all
frontier-all: frontier-linux-all frontier-darwin-all frontier-windows-all

.PHONY: frontier-linux-all
frontier-linux-all:
	@echo "Building frontier for Linux..."
	$(call build-platform,linux,amd64,linux,frontier,cmd/frontier/main.go)
	$(call build-platform,linux,arm64,linux,frontier,cmd/frontier/main.go)

.PHONY: frontier-darwin-all
frontier-darwin-all:
	@echo "Building frontier for macOS..."
	$(call build-platform,darwin,amd64,darwin,frontier,cmd/frontier/main.go)
	$(call build-platform,darwin,arm64,darwin,frontier,cmd/frontier/main.go)

.PHONY: frontier-windows-all
frontier-windows-all:
	@echo "Building frontier for Windows..."
	$(call build-platform,windows,amd64,windows,frontier,cmd/frontier/main.go)
	$(call build-platform,windows,arm64,windows,frontier,cmd/frontier/main.go)

# Cross-compilation targets for frontlas
.PHONY: frontlas-all
frontlas-all: frontlas-linux-all frontlas-darwin-all frontlas-windows-all

.PHONY: frontlas-linux-all
frontlas-linux-all:
	@echo "Building frontlas for Linux..."
	$(call build-platform,linux,amd64,linux,frontlas,cmd/frontlas/main.go)
	$(call build-platform,linux,arm64,linux,frontlas,cmd/frontlas/main.go)

.PHONY: frontlas-darwin-all
frontlas-darwin-all:
	@echo "Building frontlas for macOS..."
	$(call build-platform,darwin,amd64,darwin,frontlas,cmd/frontlas/main.go)
	$(call build-platform,darwin,arm64,darwin,frontlas,cmd/frontlas/main.go)

.PHONY: frontlas-windows-all
frontlas-windows-all:
	@echo "Building frontlas for Windows..."
	$(call build-platform,windows,amd64,windows,frontlas,cmd/frontlas/main.go)
	$(call build-platform,windows,arm64,windows,frontlas,cmd/frontlas/main.go)

# Build all binaries for all platforms
.PHONY: build-all
build-all: frontier-all frontlas-all
	@echo ""
	@echo "=========================================="
	@echo "All binaries built successfully!"
	@echo "Output directory: $(DIST_DIR)"
	@echo "=========================================="
	@echo ""
	@echo "Built binaries:"
	@ls -lh $(DIST_DIR)/* 2>/dev/null | awk '{print "  " $$9}' || \
	 find $(DIST_DIR) -maxdepth 1 -type f \( -perm +111 -o -name "*.exe" \) 2>/dev/null | sort | sed 's|^|  |' || \
	 echo "  Run 'ls -lh $(DIST_DIR)' to see built binaries"

# Individual platform builds (convenience targets)
.PHONY: frontier-linux-amd64 frontier-linux-arm64
.PHONY: frontier-darwin-amd64 frontier-darwin-arm64
.PHONY: frontier-windows-amd64 frontier-windows-arm64

frontier-linux-amd64:
	$(call build-platform,linux,amd64,linux,frontier,cmd/frontier/main.go)

frontier-linux-arm64:
	$(call build-platform,linux,arm64,linux,frontier,cmd/frontier/main.go)

frontier-darwin-amd64:
	$(call build-platform,darwin,amd64,darwin,frontier,cmd/frontier/main.go)

frontier-darwin-arm64:
	$(call build-platform,darwin,arm64,darwin,frontier,cmd/frontier/main.go)

frontier-windows-amd64:
	$(call build-platform,windows,amd64,windows,frontier,cmd/frontier/main.go)

frontier-windows-arm64:
	$(call build-platform,windows,arm64,windows,frontier,cmd/frontier/main.go)

.PHONY: frontlas-linux-amd64 frontlas-linux-arm64
.PHONY: frontlas-darwin-amd64 frontlas-darwin-arm64
.PHONY: frontlas-windows-amd64 frontlas-windows-arm64

frontlas-linux-amd64:
	$(call build-platform,linux,amd64,linux,frontlas,cmd/frontlas/main.go)

frontlas-linux-arm64:
	$(call build-platform,linux,arm64,linux,frontlas,cmd/frontlas/main.go)

frontlas-darwin-amd64:
	$(call build-platform,darwin,amd64,darwin,frontlas,cmd/frontlas/main.go)

frontlas-darwin-arm64:
	$(call build-platform,darwin,arm64,darwin,frontlas,cmd/frontlas/main.go)

frontlas-windows-amd64:
	$(call build-platform,windows,amd64,windows,frontlas,cmd/frontlas/main.go)

frontlas-windows-arm64:
	$(call build-platform,windows,arm64,windows,frontlas,cmd/frontlas/main.go)

# example
.PHONY: examples
examples:
	make -C examples
	mv examples/iclm/bin/* ./bin/ && rm -rf examples/iclm/bin
	mv examples/chatroom/bin/* ./bin/ && rm -rf examples/chatroom/bin
	mv examples/rtmp/bin/* ./bin/ && rm -rf examples/rtmp/bin

# clean
.PHONY: clean
clean:
	rm -f ./bin/* || true
	rm -rf $(DIST_DIR) || true
	make clean -C examples
	make clean -C test/bench

.PHONY: clean-dist
clean-dist:
	rm -rf $(DIST_DIR) || true

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

# Docker-based cross-compilation
# These targets use Docker to build binaries, avoiding CGO cross-compilation issues
# Prerequisites: Docker with buildx support
# To enable buildx: docker buildx create --use --name builder
DOCKER_BUILDX := docker buildx build
DOCKER_BUILD_FLAGS := --platform

# Check if Docker is available
.PHONY: check-docker
check-docker:
	@command -v docker >/dev/null 2>&1 || { echo "Error: docker is not installed"; exit 1; }
	@docker buildx version >/dev/null 2>&1 || { echo "Error: docker buildx is not available. Run: docker buildx create --use --name builder"; exit 1; }
	@echo "Docker and buildx are available"

# Helper function for Docker builds
# Binary files are named as: {binary}-{os}-{arch} or {binary}-{os}-{arch}.exe (Windows)
define docker-build-platform
	@echo "Building $(1)/$(2) using Docker..."
	@mkdir -p $(DIST_DIR)
	@if [ "$(3)" = "windows" ]; then \
		$(DOCKER_BUILDX) $(DOCKER_BUILD_FLAGS) $(1)/$(2) \
			--build-arg TARGETOS=$(1) \
			--build-arg TARGETARCH=$(2) \
			--build-arg BINARY_NAME=$(4) \
			--build-arg BUILD_PATH=$(5) \
			--target output \
			--output type=local,dest=$(DIST_DIR)/tmp-$(3)-$(2) \
			-f images/Dockerfile.build . && \
		mv $(DIST_DIR)/tmp-$(3)-$(2)/binary $(DIST_DIR)/$(4)-$(3)-$(2).exe 2>/dev/null || \
		cp $(DIST_DIR)/tmp-$(3)-$(2)/binary $(DIST_DIR)/$(4)-$(3)-$(2).exe && \
		rm -rf $(DIST_DIR)/tmp-$(3)-$(2); \
	else \
		$(DOCKER_BUILDX) $(DOCKER_BUILD_FLAGS) $(1)/$(2) \
			--build-arg TARGETOS=$(1) \
			--build-arg TARGETARCH=$(2) \
			--build-arg BINARY_NAME=$(4) \
			--build-arg BUILD_PATH=$(5) \
			--target output \
			--output type=local,dest=$(DIST_DIR)/tmp-$(3)-$(2) \
			-f images/Dockerfile.build . && \
		mv $(DIST_DIR)/tmp-$(3)-$(2)/binary $(DIST_DIR)/$(4)-$(3)-$(2) 2>/dev/null || \
		cp $(DIST_DIR)/tmp-$(3)-$(2)/binary $(DIST_DIR)/$(4)-$(3)-$(2) && \
		rm -rf $(DIST_DIR)/tmp-$(3)-$(2); \
	fi
endef

# Docker-based cross-compilation targets for frontier
.PHONY: docker-frontier-all
docker-frontier-all: check-docker docker-frontier-linux-all docker-frontier-darwin-all docker-frontier-windows-all

.PHONY: docker-frontier-linux-all
docker-frontier-linux-all:
	@echo "Building frontier for Linux using Docker..."
	$(call docker-build-platform,linux,amd64,linux,frontier,cmd/frontier/main.go)
	$(call docker-build-platform,linux,arm64,linux,frontier,cmd/frontier/main.go)

.PHONY: docker-frontier-darwin-all
docker-frontier-darwin-all:
	@echo "Building frontier for macOS using Docker..."
	$(call docker-build-platform,darwin,amd64,darwin,frontier,cmd/frontier/main.go)
	$(call docker-build-platform,darwin,arm64,darwin,frontier,cmd/frontier/main.go)

.PHONY: docker-frontier-windows-all
docker-frontier-windows-all:
	@echo "Building frontier for Windows using Docker..."
	$(call docker-build-platform,windows,amd64,windows,frontier,cmd/frontier/main.go)
	$(call docker-build-platform,windows,arm64,windows,frontier,cmd/frontier/main.go)

# Docker-based cross-compilation targets for frontlas
.PHONY: docker-frontlas-all
docker-frontlas-all: check-docker docker-frontlas-linux-all docker-frontlas-darwin-all docker-frontlas-windows-all

.PHONY: docker-frontlas-linux-all
docker-frontlas-linux-all:
	@echo "Building frontlas for Linux using Docker..."
	$(call docker-build-platform,linux,amd64,linux,frontlas,cmd/frontlas/main.go)
	$(call docker-build-platform,linux,arm64,linux,frontlas,cmd/frontlas/main.go)

.PHONY: docker-frontlas-darwin-all
docker-frontlas-darwin-all:
	@echo "Building frontlas for macOS using Docker..."
	$(call docker-build-platform,darwin,amd64,darwin,frontlas,cmd/frontlas/main.go)
	$(call docker-build-platform,darwin,arm64,darwin,frontlas,cmd/frontlas/main.go)

.PHONY: docker-frontlas-windows-all
docker-frontlas-windows-all:
	@echo "Building frontlas for Windows using Docker..."
	$(call docker-build-platform,windows,amd64,windows,frontlas,cmd/frontlas/main.go)
	$(call docker-build-platform,windows,arm64,windows,frontlas,cmd/frontlas/main.go)

# Build all binaries using Docker
.PHONY: docker-build-all
docker-build-all: check-docker docker-frontier-all docker-frontlas-all
	@echo ""
	@echo "=========================================="
	@echo "All binaries built successfully using Docker!"
	@echo "Output directory: $(DIST_DIR)"
	@echo "=========================================="
	@echo ""
	@echo "Built binaries:"
	@ls -lh $(DIST_DIR)/* 2>/dev/null | awk '{print "  " $$9}' || \
	 find $(DIST_DIR) -maxdepth 1 -type f \( -perm +111 -o -name "*.exe" \) 2>/dev/null | sort | sed 's|^|  |' || \
	 echo "  Run 'ls -lh $(DIST_DIR)' to see built binaries"

# Individual Docker build targets (convenience)
.PHONY: docker-frontier-linux-amd64 docker-frontier-linux-arm64
.PHONY: docker-frontier-darwin-amd64 docker-frontier-darwin-arm64
.PHONY: docker-frontier-windows-amd64 docker-frontier-windows-arm64

docker-frontier-linux-amd64:
	$(call docker-build-platform,linux,amd64,linux,frontier,cmd/frontier/main.go)

docker-frontier-linux-arm64:
	$(call docker-build-platform,linux,arm64,linux,frontier,cmd/frontier/main.go)

docker-frontier-darwin-amd64:
	$(call docker-build-platform,darwin,amd64,darwin,frontier,cmd/frontier/main.go)

docker-frontier-darwin-arm64:
	$(call docker-build-platform,darwin,arm64,darwin,frontier,cmd/frontier/main.go)

docker-frontier-windows-amd64:
	$(call docker-build-platform,windows,amd64,windows,frontier,cmd/frontier/main.go)

docker-frontier-windows-arm64:
	$(call docker-build-platform,windows,arm64,windows,frontier,cmd/frontier/main.go)

.PHONY: docker-frontlas-linux-amd64 docker-frontlas-linux-arm64
.PHONY: docker-frontlas-darwin-amd64 docker-frontlas-darwin-arm64
.PHONY: docker-frontlas-windows-amd64 docker-frontlas-windows-arm64

docker-frontlas-linux-amd64:
	$(call docker-build-platform,linux,amd64,linux,frontlas,cmd/frontlas/main.go)

docker-frontlas-linux-arm64:
	$(call docker-build-platform,linux,arm64,linux,frontlas,cmd/frontlas/main.go)

docker-frontlas-darwin-amd64:
	$(call docker-build-platform,darwin,amd64,darwin,frontlas,cmd/frontlas/main.go)

docker-frontlas-darwin-arm64:
	$(call docker-build-platform,darwin,arm64,darwin,frontlas,cmd/frontlas/main.go)

docker-frontlas-windows-amd64:
	$(call docker-build-platform,windows,amd64,windows,frontlas,cmd/frontlas/main.go)

docker-frontlas-windows-arm64:
	$(call docker-build-platform,windows,arm64,windows,frontlas,cmd/frontlas/main.go)
.PHONY: install-systemd
install-systemd: frontier
	@echo "Installing frontier systemd service..."
	@if [ "$(shell id -u)" -ne 0 ]; then \
		echo "Error: This target requires root privileges. Please run with sudo."; \
		exit 1; \
	fi
	./dist/systemd/install.sh

.PHONY: uninstall-systemd
uninstall-systemd:
	@echo "Uninstalling frontier systemd service..."
	@if [ "$(shell id -u)" -ne 0 ]; then \
		echo "Error: This target requires root privileges. Please run with sudo."; \
		exit 1; \
	fi
	./dist/systemd/uninstall.sh

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


