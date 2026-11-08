package edgebound

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	gconfig "github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/frontier/config"
)

func TestDynamicTLSConfigurationAndShutdown(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	conf := &config.Configuration{}
	conf.Daemon.FrontierID = "test"
	conf.Edgebound.Listen = gconfig.Listen{Network: "tcp", Addr: "127.0.0.1:0", TLS: gconfig.TLS{Enable: true, MTLS: true, Source: "callback", Callback: gconfig.TLSCallback{Service: "cert", ProviderKeys: []string{base64.StdEncoding.EncodeToString(pub)}}}}
	ln, err := listenEdge(conf, nil)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- ln.Close() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
	}
	conf.Edgebound.Listen.TLS.CACerts = []string{"file.pem"}
	if _, err = listenEdge(conf, nil); err == nil {
		t.Fatal("mixed sources accepted")
	}
	conf.Edgebound.Listen.TLS.CACerts = nil
	conf.Edgebound.Listen.TLS.Callback.ProviderKeys = nil
	if _, err = listenEdge(conf, nil); err == nil {
		t.Fatal("unauthenticated provider accepted")
	}
}
