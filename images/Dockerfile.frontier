FROM golang:1.22.0 AS builder

# an automatic platform ARG enabled by Docker BuildKit.
ARG TARGETOS
# an automatic platform ARG enabled by Docker BuildKit.
ARG TARGETARCH

ENV GO111MODULE=on \
    GOPROXY=https://goproxy.io

WORKDIR /go/src/github.com/singchia/frontier
RUN --mount=type=bind,readwrite,target=/go/src/github.com/singchia/frontier \
    make DESTDIR=/tmp/install all install

FROM alpine:3.14

COPY --from=builder /tmp/install/ /

RUN apk --no-cache add ca-certificates wget
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
RUN wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.34-r0/glibc-2.34-r0.apk
RUN apk add glibc-2.34-r0.apk

EXPOSE 2431
EXPOSE 2432

ENTRYPOINT ["/usr/bin/frontier"]
CMD ["--config", "/usr/conf/config.yaml"]