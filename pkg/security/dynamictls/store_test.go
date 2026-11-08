package dynamictls

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"sync"
	"testing"
	"time"

	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
)

type caFixture struct {
	cert *x509.Certificate
	key  ed25519.PrivateKey
	pem  []byte
}

func newCA(t *testing.T, name string) caFixture {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: name}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return caFixture{cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}
func (c caFixture) issue(t *testing.T, pub interface{}, client bool) []byte {
	t.Helper()
	usage := x509.ExtKeyUsageServerAuth
	if client {
		usage = x509.ExtKeyUsageClientAuth
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), DNSNames: []string{"localhost"}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, pub, c.key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
func TestDynamicTLSMultipleCAsAndRotation(t *testing.T) {
	provider, key, _ := ed25519.GenerateKey(rand.Reader)
	s, err := New("frontier-a", []ed25519.PublicKey{provider}, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.TLSConfig().GetConfigForClient(nil); err == nil {
		t.Fatal("uninitialized TLS accepted")
	}
	serverCA := newCA(t, "server")
	a := newCA(t, "a")
	b := newCA(t, "b")
	outsider := newCA(t, "outside")
	revision := uint64(1)
	trust := [][]byte{a.pem, b.pem}
	var serverCert []byte
	fetch := func(_ context.Context, req securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
		block, _ := pem.Decode(req.CSR)
		csr, err := x509.ParseCertificateRequest(block.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if err = csr.CheckSignature(); err != nil {
			t.Fatal(err)
		}
		if serverCert == nil {
			serverCert = serverCA.issue(t, csr.PublicKey, false)
		}
		return securityv1.SignTLSConfig(req, securityv1.TLSConfig{Revision: revision, CertificateChain: serverCert, TrustedCAs: trust, ValidUntil: time.Now().Add(5 * time.Minute).Unix()}, key)
	}
	if err = s.Refresh(context.Background(), fetch); err != nil {
		t.Fatal(err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", s.TLSConfig())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(2 * time.Second))
				if conn.(*tls.Conn).Handshake() == nil {
					conn.Write([]byte{1})
				}
			}()
		}
	}()
	t.Cleanup(func() { ln.Close(); <-done; wg.Wait() })
	connect := func(c caFixture) error {
		pub, private, _ := ed25519.GenerateKey(rand.Reader)
		der, _ := x509.MarshalPKCS8PrivateKey(private)
		cert, err := tls.X509KeyPair(c.issue(t, pub, true), pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
		if err != nil {
			return err
		}
		roots := x509.NewCertPool()
		roots.AddCert(serverCA.cert)
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: time.Second}, "tcp", ln.Addr().String(), &tls.Config{RootCAs: roots, ServerName: "localhost", Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(time.Second))
		_, err = conn.Read(make([]byte, 1))
		return err
	}
	for _, ca := range []caFixture{a, b} {
		if err = connect(ca); err != nil {
			t.Fatal("trusted CA rejected", err)
		}
	}
	if connect(outsider) == nil {
		t.Fatal("untrusted CA accepted")
	}
	var concurrent sync.WaitGroup
	for i := 0; i < 8; i++ {
		concurrent.Add(1)
		go func() {
			defer concurrent.Done()
			for j := 0; j < 5; j++ {
				if e := connect(b); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	for i := 0; i < 5; i++ {
		if err = s.Refresh(context.Background(), fetch); err != nil {
			t.Fatal(err)
		}
	}
	concurrent.Wait()
	old := s.current.Load()
	// Removing A changes only future handshakes and does not mutate the old snapshot.
	revision = 2
	trust = [][]byte{b.pem}
	if err = s.Refresh(context.Background(), fetch); err != nil {
		t.Fatal(err)
	}
	if len(old.config.ClientCAs.Subjects()) != 2 {
		t.Fatal("old configuration mutated")
	}
	if connect(a) == nil {
		t.Fatal("removed CA accepted")
	}
	if err = connect(b); err != nil {
		t.Fatal(err)
	}
	revision = 1
	if s.Refresh(context.Background(), fetch) == nil {
		t.Fatal("rollback accepted")
	}
	if s.current.Load().revision != 2 {
		t.Fatal("bad update replaced snapshot")
	}
	if err = s.Refresh(context.Background(), func(context.Context, securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
		return securityv1.SignedTLSConfig{}, errors.New("offline")
	}); err == nil {
		t.Fatal("expected failure")
	}
	if err = connect(b); err != nil {
		t.Fatal("valid cache lost", err)
	}
	expired := *s.current.Load()
	expired.expires = time.Now().Add(-time.Second)
	s.current.Store(&expired)
	if connect(b) == nil {
		t.Fatal("expired configuration accepted")
	}
}
func TestDynamicTLSRejectsReplayAndForeignSignature(t *testing.T) {
	pub, key, _ := ed25519.GenerateKey(rand.Reader)
	_, foreign, _ := ed25519.GenerateKey(rand.Reader)
	req := securityv1.TLSConfigRequest{FrontierID: "a", Nonce: "nonce", CSR: []byte("csr")}
	signed, err := securityv1.SignTLSConfig(req, securityv1.TLSConfig{Revision: 1}, key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = securityv1.VerifyTLSConfig(req, signed, []ed25519.PublicKey{pub}); err != nil {
		t.Fatal(err)
	}
	changed := req
	changed.Nonce = "other"
	if _, err = securityv1.VerifyTLSConfig(changed, signed, []ed25519.PublicKey{pub}); err == nil {
		t.Fatal("replay accepted")
	}
	signed, err = securityv1.SignTLSConfig(req, securityv1.TLSConfig{Revision: 1}, foreign)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = securityv1.VerifyTLSConfig(req, signed, []ed25519.PublicKey{pub}); err == nil {
		t.Fatal("foreign provider accepted")
	}
}

func TestDynamicTLSWorkerWakeAndCancellation(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	store, err := New("frontier", []ed25519.PublicKey{pub}, true)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{}, 4)
	wake := make(chan struct{}, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		store.Run(ctx, func(ctx context.Context, _ securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
			entered <- struct{}{}
			return securityv1.SignedTLSConfig{}, errors.New("not registered")
		}, time.Minute, time.Second, func(error) {}, wake)
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("initial refresh missing")
	}
	wake <- struct{}{}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("provider notification ignored")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker leaked")
	}
}
