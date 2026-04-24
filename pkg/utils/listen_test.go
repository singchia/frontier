package utils

import (
	"testing"

	"github.com/singchia/frontier/pkg/config"
)

func TestListenUDP(t *testing.T) {
	listen := &config.Listen{
		Network: "udp",
		Addr:    "127.0.0.1:8080",
	}
	ln, err := Listen(listen)
	if err != nil {
		t.Fatalf("listen err: %s", err)
	}
	t.Logf("listen: %s", ln.Addr())
	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("accept err: %s", err)
	}
	t.Logf("conn: %s", conn.RemoteAddr())
}
