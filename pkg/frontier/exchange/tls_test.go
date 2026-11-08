package exchange

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	clusterv1 "github.com/singchia/frontier/api/controlplane/frontlas/v1"
	"google.golang.org/grpc"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/stretchr/testify/require"
)

func TestDynamicTLSCallbackOverGeminio(t *testing.T) {
	h := newHarness(t)
	svc, err := service.NewService(svcDial(), service.OptionServiceName("cert"), service.OptionServiceRegistrationConfirmation(true))
	require.NoError(t, err)
	defer svc.Close()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = service.RegisterGetTLSConfig(ctx, svc, func(callCtx context.Context, req securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
		if callCtx.Err() != nil {
			return securityv1.SignedTLSConfig{}, callCtx.Err()
		}
		return securityv1.SignTLSConfig(req, securityv1.TLSConfig{Revision: 7}, key)
	})
	require.NoError(t, err)
	// Canceling the registration context must not cancel future callbacks.
	cancel()
	ex := &exchange{Servicebound: h.sb.(apis.Servicebound)}
	call, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	req := securityv1.TLSConfigRequest{FrontierID: "frontier-a", Nonce: "0123456789012345678901234567890123456789012345678901234567890123", CSR: []byte("test-only-csr")}
	signed, err := ex.GetTLSConfig(call, "cert", req, []ed25519.PublicKey{pub})
	require.NoError(t, err)
	conf, err := securityv1.VerifyTLSConfig(req, signed, []ed25519.PublicKey{pub})
	require.NoError(t, err)
	require.Equal(t, uint64(7), conf.Revision)
	// An unrelated service name cannot select this provider.
	_, err = ex.GetTLSConfig(call, "other", req, []ed25519.PublicKey{pub})
	require.Error(t, err)
}

type tlsDiscovery struct {
	clusterv1.UnimplementedClusterServiceServer
	mu        sync.RWMutex
	frontiers []*clusterv1.Frontier
}

func (d *tlsDiscovery) ListFrontiers(context.Context, *clusterv1.ListFrontiersRequest) (*clusterv1.ListFrontiersResponse, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return &clusterv1.ListFrontiersResponse{Frontiers: append([]*clusterv1.Frontier(nil), d.frontiers...)}, nil
}
func TestDynamicTLSNewFrontierDiscoveredAndRegistered(t *testing.T) {
	h := newHarness(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	discovery := &tlsDiscovery{}
	server := grpc.NewServer()
	clusterv1.RegisterClusterServiceServer(server, discovery)
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	defer func() { server.Stop(); <-served }()
	svc, err := service.NewClusterService(listener.Addr().String(), service.OptionServiceName("cert"), service.OptionServiceRegistrationConfirmation(true))
	require.NoError(t, err)
	defer svc.Close()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	err = service.RegisterGetTLSConfig(context.Background(), svc, func(ctx context.Context, req securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error) {
		return securityv1.SignTLSConfig(req, securityv1.TLSConfig{Revision: 9}, key)
	})
	require.NoError(t, err)
	discovery.mu.Lock()
	discovery.frontiers = []*clusterv1.Frontier{{FrontierId: "new-frontier", AdvertisedSbAddr: serviceboundAddr}}
	discovery.mu.Unlock()
	ex := &exchange{Servicebound: h.sb.(apis.Servicebound)}
	req := securityv1.TLSConfigRequest{FrontierID: "new-frontier", Nonce: strings.Repeat("a", 64), CSR: []byte("test-only-csr")}
	deadline := time.Now().Add(14 * time.Second)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		response, e := ex.GetTLSConfig(ctx, "cert", req, []ed25519.PublicKey{pub})
		cancel()
		if e == nil {
			cfg, e := securityv1.VerifyTLSConfig(req, response, []ed25519.PublicKey{pub})
			require.NoError(t, e)
			require.Equal(t, uint64(9), cfg.Revision)
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("new Frontier callback was not discovered", e)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
