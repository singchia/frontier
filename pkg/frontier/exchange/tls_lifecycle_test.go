package exchange

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	gconfig "github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/edgebound"
	"github.com/singchia/frontier/pkg/frontier/mq"
	"github.com/singchia/frontier/pkg/frontier/repo"
	"github.com/singchia/frontier/pkg/frontier/servicebound"
	"github.com/singchia/geminio"
	"github.com/singchia/go-timer/v2"
	"github.com/stretchr/testify/require"
)

type lifecycleCA struct {
	cert   *x509.Certificate
	key    ed25519.PrivateKey
	bundle []byte
}

func lifecycleNewCA(t *testing.T, name string) lifecycleCA {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	template := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: name}, IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, pub, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return lifecycleCA{cert, key, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}
func (c lifecycleCA) issue(pub interface{}, serial int64, usage x509.ExtKeyUsage) ([]byte, error) {
	template := &x509.Certificate{SerialNumber: big.NewInt(serial), DNSNames: []string{"localhost"}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, c.cert, pub, c.key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), nil
}
func lifecycleAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

func TestDynamicTLSFullLifecycle(t *testing.T) {
	serverCA := lifecycleNewCA(t, "server")
	a := lifecycleNewCA(t, "edge-a")
	b := lifecycleNewCA(t, "edge-b")
	foreign := lifecycleNewCA(t, "foreign")
	providerPub, providerKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sbAddr, ebAddr := lifecycleAddr(t), lifecycleAddr(t)
	conf := &config.Configuration{}
	conf.Daemon.FrontierID = "lifecycle-frontier"
	conf.Servicebound.Listen = gconfig.Listen{Network: "tcp", Addr: sbAddr}
	conf.Edgebound.Listen = gconfig.Listen{Network: "tcp", Addr: ebAddr, TLS: gconfig.TLS{Enable: true, MTLS: true, Source: "callback", Callback: gconfig.TLSCallback{Service: "cert", ProviderKeys: []string{base64.StdEncoding.EncodeToString(providerPub)}, RefreshInterval: "1s", Timeout: "500ms"}}}
	repository, err := repo.NewRepo(conf)
	require.NoError(t, err)
	mqm, err := mq.NewMQM(conf)
	require.NoError(t, err)
	tmr := timer.NewTimer()
	ex := NewExchange(conf, mqm)
	sb, err := servicebound.NewServicebound(conf, repository, nil, ex, mqm, tmr)
	require.NoError(t, err)
	eb, err := edgebound.NewEdgebound(conf, repository, nil, ex, tmr)
	require.NoError(t, err)
	sbDone, ebDone := make(chan error, 1), make(chan error, 1)
	go func() { sbDone <- sb.Serve() }()
	go func() { ebDone <- eb.Serve() }()
	t.Cleanup(func() {
		eb.Close()
		sb.Close()
		<-ebDone
		<-sbDone
		repository.Close()
		mqm.Close()
		tmr.Close()
	})
	dialService := func() (net.Conn, error) { return net.DialTimeout("tcp", sbAddr, time.Second) }
	business, err := service.NewService(dialService, service.OptionServiceName("business"), service.OptionServiceRegistrationConfirmation(true))
	require.NoError(t, err)
	defer business.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var nextID atomic.Uint64
	nextID.Store(100)
	require.NoError(t, business.RegisterGetEdgeID(ctx, func([]byte) (uint64, error) { return nextID.Add(1), nil }))
	require.NoError(t, business.Register(ctx, "echo", func(_ context.Context, req geminio.Request, rsp geminio.Response) { rsp.SetData(req.Data()) }))
	roots := x509.NewCertPool()
	roots.AddCert(serverCA.cert)
	makeClient := func(c lifecycleCA) tls.Certificate {
		pub, key, e := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, e)
		cert, e := c.issue(pub, time.Now().UnixNano(), x509.ExtKeyUsageClientAuth)
		require.NoError(t, e)
		der, e := x509.MarshalPKCS8PrivateKey(key)
		require.NoError(t, e)
		pair, e := tls.X509KeyPair(cert, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
		require.NoError(t, e)
		return pair
	}
	clientA, clientB, clientForeign := makeClient(a), makeClient(b), makeClient(foreign)
	connect := func(cert tls.Certificate) (edge.Edge, int64, error) {
		var cn *tls.Conn
		var serial int64
		e, err := edge.NewNoRetryEdge(func() (net.Conn, error) {
			raw, err := net.DialTimeout("tcp", ebAddr, time.Second)
			if err != nil {
				return nil, err
			}
			raw.SetDeadline(time.Now().Add(2 * time.Second))
			cn = tls.Client(raw, &tls.Config{MinVersion: tls.VersionTLS12, RootCAs: roots, ServerName: "localhost", Certificates: []tls.Certificate{cert}})
			if err = cn.Handshake(); err != nil {
				cn.Close()
				return nil, err
			}
			serial = cn.ConnectionState().PeerCertificates[0].SerialNumber.Int64()
			return cn, nil
		})
		if err != nil {
			if cn != nil {
				cn.Close()
			}
			return nil, serial, err
		}
		cn.SetDeadline(time.Time{})
		return e, serial, nil
	}
	echo := func(e edge.Edge) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		rsp, err := e.Call(ctx, "echo", e.NewRequest([]byte("probe")))
		if err != nil {
			return err
		}
		if rsp.Error() != nil {
			return rsp.Error()
		}
		if string(rsp.Data()) != "probe" {
			return errors.New("unexpected echo")
		}
		return nil
	}
	e, _, err := connect(clientA)
	if e != nil {
		e.Close()
	}
	require.Error(t, err, "must not accept Edge before initial configuration")
	var mu sync.Mutex
	revision := uint64(1)
	trusted := [][]byte{a.bundle, b.bundle}
	chainByRevision := make(map[uint64][]byte)
	var calls atomic.Uint64
	startProvider := func() service.Service {
		svc, e := service.NewService(dialService, service.OptionServiceName("cert"), service.OptionServiceRegistrationConfirmation(true))
		require.NoError(t, e)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		e = service.RegisterGetTLSConfig(ctx, svc, func(ctx context.Context, req securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
			calls.Add(1)
			mu.Lock()
			defer mu.Unlock()
			if req.FrontierID != "lifecycle-frontier" {
				return securityv1.SignedTLSConfig{}, errors.New("unknown frontier")
			}
			if chainByRevision[revision] == nil {
				block, _ := pem.Decode(req.CSR)
				if block == nil {
					return securityv1.SignedTLSConfig{}, errors.New("missing CSR")
				}
				csr, e := x509.ParseCertificateRequest(block.Bytes)
				if e != nil {
					return securityv1.SignedTLSConfig{}, e
				}
				if e = csr.CheckSignature(); e != nil {
					return securityv1.SignedTLSConfig{}, e
				}
				chain, e := serverCA.issue(csr.PublicKey, int64(revision), x509.ExtKeyUsageServerAuth)
				if e != nil {
					return securityv1.SignedTLSConfig{}, e
				}
				chainByRevision[revision] = chain
			}
			return securityv1.SignTLSConfig(req, securityv1.TLSConfig{Revision: revision, CertificateChain: chainByRevision[revision], TrustedCAs: trusted, ValidUntil: time.Now().Add(6 * time.Second).Unix()}, providerKey)
		})
		require.NoError(t, e)
		return svc
	}
	provider := startProvider()
	defer func() {
		if provider != nil {
			provider.Close()
		}
	}()
	waitConnect := func(cert tls.Certificate, wanted int64) edge.Edge {
		t.Helper()
		until := time.Now().Add(8 * time.Second)
		attempt := 0
		for {
			attempt++
			e, serial, err := connect(cert)
			if attempt == 1 && err != nil {
				t.Logf("first connection failure serial=%d: %v", serial, err)
			}
			if err == nil && serial == wanted {
				return e
			}
			if e != nil {
				e.Close()
			}
			if time.Now().After(until) {
				t.Fatalf("certificate %d not ready: serial=%d err=%v", wanted, serial, err)
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
	oldEdge := waitConnect(clientA, 1)
	defer oldEdge.Close()
	require.NoError(t, echo(oldEdge))
	second := waitConnect(clientB, 1)
	require.NoError(t, echo(second))
	second.Close()
	invalid, _, err := connect(clientForeign)
	if invalid != nil {
		invalid.Close()
	}
	require.Error(t, err)
	t.Log("initial callback and multi-CA admission succeeded; unrelated CA rejected")
	mu.Lock()
	revision = 2
	trusted = [][]byte{b.bundle}
	mu.Unlock()
	fresh := waitConnect(clientB, 2)
	require.NoError(t, echo(fresh))
	fresh.Close()
	removed, _, err := connect(clientA)
	if removed != nil {
		removed.Close()
	}
	require.Error(t, err)
	require.NoError(t, echo(oldEdge), "normal rotation must retain existing connection")
	t.Log("server certificate rotated to serial 2; removed CA rejected; existing connection survived")
	closing := time.Now()
	require.NoError(t, provider.Close())
	t.Logf("provider close took %s; business echo=%v", time.Since(closing), echo(oldEdge))
	provider = nil
	cached := waitConnect(clientB, 2)
	require.NoError(t, echo(cached))
	cached.Close()
	until := time.Now().Add(9 * time.Second)
	for {
		e, _, err := connect(clientB)
		if e != nil {
			e.Close()
		}
		if err != nil {
			break
		}
		if time.Now().After(until) {
			t.Fatal("expired snapshot still accepted new connection")
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NoError(t, echo(oldEdge))
	t.Log("provider disconnected: valid cache worked; lease expiry rejected new connections")
	provider = startProvider()
	recovered := waitConnect(clientB, 2)
	require.NoError(t, echo(recovered))
	recovered.Close()
	t.Log("provider reconnected: callback registration restored TLS admission")
	var wg sync.WaitGroup
	failures := make(chan error, 64)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 8; j++ {
				e, _, err := connect(clientB)
				if err == nil {
					err = echo(e)
					e.Close()
				}
				if err != nil {
					failures <- err
				}
			}
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
	t.Logf("64 short Edge connections + RPC echoes completed; callback calls=%d", calls.Load())
}
