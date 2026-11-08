// Package dynamictls maintains verified immutable TLS snapshots.
package dynamictls

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
)

type Fetch func(context.Context, securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error)
type snapshot struct {
	config   *tls.Config
	revision uint64
	expires  time.Time
	chain    []byte
	cas      [][]byte
}
type Store struct {
	id        string
	key       ed25519.PrivateKey
	csr       []byte
	providers []ed25519.PublicKey
	mtls      bool
	current   atomic.Pointer[snapshot]
	refresh   atomic.Bool
}

func New(id string, providers []ed25519.PublicKey, mtls bool) (*Store, error) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return NewWithKey(id, providers, mtls, key)
}

// NewWithKey uses a provisioned local key so enrollment survives process restart.
func NewWithKey(id string, providers []ed25519.PublicKey, mtls bool, key ed25519.PrivateKey) (*Store, error) {
	if len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid local private key")
	}
	key = append(ed25519.PrivateKey(nil), key...)
	if id == "" || len(providers) == 0 {
		return nil, errors.New("Frontier identity and provider keys are required")
	}
	keys := make([]ed25519.PublicKey, len(providers))
	for i, k := range providers {
		if len(k) != ed25519.PublicKeySize {
			return nil, errors.New("invalid provider public key")
		}
		keys[i] = append(ed25519.PublicKey(nil), k...)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{}, key)
	if err != nil {
		return nil, err
	}
	return &Store{id: id, key: key, csr: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr}), providers: keys, mtls: mtls}, nil
}
func (s *Store) TLSConfig() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS12, SessionTicketsDisabled: true, GetConfigForClient: func(*tls.ClientHelloInfo) (*tls.Config, error) {
		v := s.current.Load()
		if v == nil || !time.Now().Before(v.expires) {
			return nil, errors.New("TLS configuration unavailable or expired")
		}
		return v.config, nil
	}}
}
func (s *Store) Refresh(ctx context.Context, fetch Fetch) error {
	if !s.refresh.CompareAndSwap(false, true) {
		return errors.New("TLS refresh already in progress")
	}
	defer s.refresh.Store(false)
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	req := securityv1.TLSConfigRequest{FrontierID: s.id, Nonce: hex.EncodeToString(nonce), CSR: append([]byte(nil), s.csr...)}
	old := s.current.Load()
	if old != nil {
		req.Revision = old.revision
	}
	signed, err := fetch(ctx, req)
	if err != nil {
		return err
	}
	conf, err := securityv1.VerifyTLSConfig(req, signed, s.providers)
	if err != nil {
		return err
	}
	if old != nil && conf.Revision < old.revision {
		return errors.New("TLS revision rollback")
	}
	// Equal revisions may extend the lease, but cannot change certificate or trust content.
	if old != nil && conf.Revision == old.revision && (!equalBytes(old.chain, conf.CertificateChain) || !equalCAs(old.cas, conf.TrustedCAs)) {
		return errors.New("TLS revision content conflict")
	}
	cfg, expires, err := s.validate(conf)
	if err != nil {
		return err
	}
	s.current.Store(&snapshot{config: cfg, revision: conf.Revision, expires: expires, chain: conf.CertificateChain, cas: conf.TrustedCAs})
	return nil
}
func equalBytes(a, b []byte) bool { return string(a) == string(b) }
func equalCAs(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !equalBytes(a[i], b[i]) {
			return false
		}
	}
	return true
}
func (s *Store) validate(c securityv1.TLSConfig) (*tls.Config, time.Time, error) {
	now := time.Now()
	expires := time.Unix(c.ValidUntil, 0)
	if c.Revision == 0 || !expires.After(now) || expires.After(now.Add(24*time.Hour)) {
		return nil, time.Time{}, errors.New("invalid TLS revision or lease (maximum 24h)")
	}
	der, err := x509.MarshalPKCS8PrivateKey(s.key)
	if err != nil {
		return nil, time.Time{}, err
	}
	pair, err := tls.X509KeyPair(c.CertificateChain, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	if err != nil {
		return nil, time.Time{}, errors.New("certificate does not match local key")
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return nil, time.Time{}, err
	}
	if now.Before(leaf.NotBefore) || !now.Before(leaf.NotAfter) || leaf.IsCA {
		return nil, time.Time{}, errors.New("invalid server certificate validity")
	}
	validUsage := false
	for _, u := range leaf.ExtKeyUsage {
		if u == x509.ExtKeyUsageServerAuth {
			validUsage = true
		}
	}
	if !validUsage {
		return nil, time.Time{}, errors.New("serverAuth certificate required")
	}
	if leaf.NotAfter.Before(expires) {
		expires = leaf.NotAfter
	}
	pair.Leaf = leaf
	roots := x509.NewCertPool()
	if len(c.TrustedCAs) > 64 {
		return nil, time.Time{}, errors.New("too many trusted CA bundles")
	}
	count := 0
	for _, bundle := range c.TrustedCAs {
		rest := bytes.TrimSpace(bundle)
		for len(rest) > 0 {
			block, tail := pem.Decode(rest)
			if block == nil || block.Type != "CERTIFICATE" {
				return nil, time.Time{}, errors.New("invalid CA PEM")
			}
			rest = bytes.TrimSpace(tail)
			ca, err := x509.ParseCertificate(block.Bytes)
			if err != nil || !ca.IsCA || !ca.BasicConstraintsValid || ca.KeyUsage&x509.KeyUsageCertSign == 0 {
				return nil, time.Time{}, errors.New("invalid CA certificate")
			}
			roots.AddCert(ca)
			count++
		}
	}
	cfg := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{pair}, SessionTicketsDisabled: true}
	if s.mtls {
		if count == 0 {
			return nil, time.Time{}, errors.New("mTLS requires trusted CAs")
		}
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		cfg.ClientCAs = roots
	}
	return cfg, expires, nil
}

// Run is owned and joined by the listener. Calls never execute on the handshake path.
func (s *Store) Run(ctx context.Context, fetch Fetch, interval, timeout time.Duration, report func(error), wake <-chan struct{}) {
	delay := time.Duration(0)
	backoff := time.Second
	var lastCall time.Time
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		case <-wake:
			timer.Stop()
		}
		// Coalesce untrusted registration wakeups; never turn them into an RPC storm.
		if wait := time.Second - time.Since(lastCall); wait > 0 {
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
		lastCall = time.Now()
		call, cancel := context.WithTimeout(ctx, timeout)
		err := s.Refresh(call, fetch)
		cancel()
		if err != nil {
			report(fmt.Errorf("refresh TLS configuration: %w", err))
			delay = jitter(backoff)
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		} else {
			backoff = time.Second
			delay = jitter(interval)
			if v := s.current.Load(); v != nil {
				remaining := time.Until(v.expires) / 2
				if remaining < delay {
					delay = remaining
				}
				if delay < time.Second {
					delay = time.Second
				}
			}
		}
	}
}
func jitter(d time.Duration) time.Duration {
	n, err := rand.Int(rand.Reader, big.NewInt(401))
	if err != nil {
		return d
	}
	return d * time.Duration(800+n.Int64()) / 1000
}
