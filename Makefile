all: frontier examples

.PHONY: frontier
frontier:
	go build -trimpath -ldflags "-s -w" -o ./frontier cmd/frontier/main.go

.PHONY: frontier-linux
frontier-linux:
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o ./frontier cmd/frontier/main.go

#docker: linux
#	docker build -t harbor.moresec.cn/moresec/ms_gw:1.4.0 .
#	docker push harbor.moresec.cn/moresec/ms_gw:1.4.0

.PHONY: examples
examples:
	make -C examples

.PHONY: clean
clean:
	rm ./frontier
	rm ./examples/iclm/iclm_edge
	rm ./examples/iclm/iclm_service

.PHONY: output
output: build


