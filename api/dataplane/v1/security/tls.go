// Package security defines the optional dynamic TLS control callback contract.
package security

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
)

const RPCGetTLSConfig = "get_tls_config_v1"
const MaxMessageSize = 1 << 20

type TLSConfigRequest struct {
	FrontierID string `json:"frontier_id"`
	Nonce      string `json:"nonce"`
	CSR        []byte `json:"csr"`
	Revision   uint64 `json:"revision"`
}
type TLSConfig struct {
	Revision         uint64   `json:"revision"`
	CertificateChain []byte   `json:"certificate_chain"`
	TrustedCAs       [][]byte `json:"trusted_cas"`
	ValidUntil       int64    `json:"valid_until"`
}
type SignedTLSConfig struct {
	Payload   []byte `json:"payload"`
	Signature []byte `json:"signature"`
}

func signatureInput(req TLSConfigRequest, payload []byte) []byte {
	b, _ := json.Marshal(req) // This concrete struct cannot fail JSON encoding.
	return append(append([]byte("frontier-tls-config-v1\x00"), b...), payload...)
}
func SignTLSConfig(req TLSConfigRequest, conf TLSConfig, key ed25519.PrivateKey) (SignedTLSConfig, error) {
	if len(key) != ed25519.PrivateKeySize {
		return SignedTLSConfig{}, errors.New("invalid signing key")
	}
	b, err := json.Marshal(conf)
	if err != nil {
		return SignedTLSConfig{}, err
	}
	if len(b) > MaxMessageSize/2 {
		return SignedTLSConfig{}, errors.New("TLS configuration too large")
	}
	return SignedTLSConfig{Payload: b, Signature: ed25519.Sign(key, signatureInput(req, b))}, nil
}
func VerifyTLSConfig(req TLSConfigRequest, signed SignedTLSConfig, keys []ed25519.PublicKey) (TLSConfig, error) {
	var conf TLSConfig
	if len(signed.Payload) > MaxMessageSize {
		return conf, errors.New("TLS configuration too large")
	}
	for _, key := range keys {
		if len(key) == ed25519.PublicKeySize && ed25519.Verify(key, signatureInput(req, signed.Payload), signed.Signature) {
			err := json.Unmarshal(signed.Payload, &conf)
			return conf, err
		}
	}
	return conf, errors.New("untrusted TLS configuration signature")
}
