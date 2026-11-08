package utils

import (
	"github.com/singchia/frontier/pkg/config"
	"testing"
)

func TestDynamicTLSSourceValidation(t *testing.T) {
	for _, source := range []string{"callback", "unknown"} {
		if ln, err := Listen(&config.Listen{Network: "tcp", Addr: "127.0.0.1:0", TLS: config.TLS{Source: source}}); err == nil {
			ln.Close()
			t.Fatal("unsupported dynamic source accepted")
		}
	}
	for _, source := range []string{"", "file"} {
		ln, err := Listen(&config.Listen{Network: "tcp", Addr: "127.0.0.1:0", TLS: config.TLS{Source: source}})
		if err != nil {
			t.Fatal(err)
		}
		if err = ln.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
