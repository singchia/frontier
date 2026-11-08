package dynamictls

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"

	security "github.com/singchia/frontier/api/dataplane/v1/security"
)

func TestProvisionedKeySurvivesRestart(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "node.key")
	if err = os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		loaded, err := ReadPrivateKey(path)
		if err != nil {
			t.Fatal(err)
		}
		store, err := NewWithKey("node", []ed25519.PublicKey{pub}, false, loaded)
		if err != nil {
			t.Fatal(err)
		}
		clear(loaded) // Constructor owns its copy.
		_ = store.Refresh(context.Background(), func(_ context.Context, r security.TLSConfigRequest) (security.SignedTLSConfig, error) {
			block, _ := pem.Decode(r.CSR)
			csr, err := x509.ParseCertificateRequest(block.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			if err = csr.CheckSignature(); err != nil {
				t.Fatal(err)
			}
			if !pub.Equal(csr.PublicKey) {
				t.Fatal("private key changed across restart")
			}
			return security.SignedTLSConfig{}, errors.New("test stops after CSR")
		})
	}
}
