package dynamictls

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// ReadPrivateKey reads a provisioned PKCS8 Ed25519 key; it never generates or uploads one.
func ReadPrivateKey(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	defer clear(b)
	block, rest := pem.Decode(b)
	if block == nil || block.Type != "PRIVATE KEY" || len(rest) != 0 {
		return nil, errors.New("invalid private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("invalid private key")
	}
	k, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, errors.New("Ed25519 private key required")
	}
	return k, nil
}
