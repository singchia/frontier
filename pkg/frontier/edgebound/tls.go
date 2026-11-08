package edgebound

import (
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/security/dynamictls"
	"github.com/singchia/frontier/pkg/utils"
	"k8s.io/klog/v2"
)

type tlsListener struct {
	net.Listener
	cancel context.CancelFunc
	done   chan struct{}
	once   sync.Once
}

func (l *tlsListener) Close() error {
	var err error
	l.once.Do(func() { l.cancel(); err = l.Listener.Close(); <-l.done })
	return err
}
func listenEdge(conf *config.Configuration, exchange apis.Exchange) (net.Listener, error) {
	listen := &conf.Edgebound.Listen
	if listen.TLS.ClientIdentityURIPrefix != "" {
		if !listen.TLS.Enable || !listen.TLS.MTLS || conf.Edgebound.BypassEnable || !strings.HasSuffix(listen.TLS.ClientIdentityURIPrefix, "/") {
			return nil, errors.New("client URI identity binding requires mTLS, no bypass and a URI prefix ending in slash")
		}
		u, err := url.Parse(listen.TLS.ClientIdentityURIPrefix)
		if err != nil || u.Scheme == "" || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
			return nil, errors.New("invalid client identity URI prefix")
		}
	}
	if listen.TLS.Source != "callback" {
		return utils.Listen(listen)
	}
	opts := listen.TLS.Callback
	if !listen.TLS.Enable || listen.Network != "tcp" || conf.Edgebound.BypassEnable || len(listen.TLS.Certs) > 0 || len(listen.TLS.CACerts) > 0 || opts.Service == "" {
		return nil, errors.New("callback TLS requires TCP, TLS enabled, provider service, no bypass and no file certificates")
	}
	interval := 60 * time.Second
	timeout := 5 * time.Second
	var err error
	if opts.RefreshInterval != "" {
		interval, err = time.ParseDuration(opts.RefreshInterval)
		if err != nil {
			return nil, err
		}
	}
	if opts.Timeout != "" {
		timeout, err = time.ParseDuration(opts.Timeout)
		if err != nil {
			return nil, err
		}
	}
	if interval < time.Second || interval > 5*time.Minute || timeout < time.Millisecond || timeout > 30*time.Second {
		return nil, errors.New("invalid callback refresh interval or timeout")
	}
	keys := []ed25519.PublicKey{}
	for _, encoded := range opts.ProviderKeys {
		k, e := base64.StdEncoding.DecodeString(encoded)
		if e != nil || len(k) != ed25519.PublicKeySize {
			return nil, errors.New("invalid provider key")
		}
		keys = append(keys, ed25519.PublicKey(k))
	}
	var store *dynamictls.Store
	if opts.PrivateKeyFile != "" {
		var key ed25519.PrivateKey
		key, err = dynamictls.ReadPrivateKey(opts.PrivateKeyFile)
		if err == nil {
			store, err = dynamictls.NewWithKey(conf.Daemon.FrontierID, keys, listen.TLS.MTLS, key)
		}
	} else {
		store, err = dynamictls.New(conf.Daemon.FrontierID, keys, listen.TLS.MTLS)
	}
	if err != nil {
		return nil, err
	}
	raw, err := net.Listen(listen.Network, listen.Addr)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	ln := &tlsListener{Listener: tls.NewListener(raw, store.TLSConfig()), cancel: cancel, done: make(chan struct{})}
	fetch := func(ctx context.Context, input securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
		provider, ok := exchange.(interface {
			GetTLSConfig(context.Context, string, securityv1.TLSConfigRequest, []ed25519.PublicKey) (securityv1.SignedTLSConfig, error)
		})
		if !ok {
			return securityv1.SignedTLSConfig{}, errors.New("TLS callback exchange unavailable")
		}
		return provider.GetTLSConfig(ctx, opts.Service, input, keys)
	}
	var wake <-chan struct{}
	if notifier, ok := exchange.(interface{ TLSConfigWake() <-chan struct{} }); ok {
		wake = notifier.TLSConfigWake()
	}
	go func() {
		defer close(ln.done)
		defer func() {
			if recover() != nil {
				klog.Error("TLS refresh worker stopped unexpectedly")
				cancel()
			}
		}()
		store.Run(ctx, fetch, interval, timeout, func(err error) { klog.Warningf("dynamic TLS refresh failed: %s", err) }, wake)
	}()
	return ln, nil
}
