package exchange

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"

	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
	"github.com/singchia/geminio/options"
)

func (ex *exchange) GetTLSConfig(ctx context.Context, provider string, input securityv1.TLSConfigRequest, keys []ed25519.PublicKey) (securityv1.SignedTLSConfig, error) {
	var result securityv1.SignedTLSConfig
	peers, err := ex.Servicebound.GetServicesByName(provider)
	if err != nil {
		return result, errors.New("TLS provider unavailable")
	}
	if len(peers) == 0 {
		return result, errors.New("TLS provider not connected")
	}
	data, err := json.Marshal(input)
	if err != nil {
		return result, err
	}
	// A bad replica cannot replace the trust root. Try remaining replicas within the call budget.
	for i, peer := range peers {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		budget := time.Second
		if deadline, ok := ctx.Deadline(); ok {
			budget = time.Until(deadline) / time.Duration(len(peers)-i)
		}
		call, cancel := context.WithTimeout(ctx, budget)
		opt := options.Call()
		opt.SetTimeout(budget)
		rsp, e := peer.Call(call, securityv1.RPCGetTLSConfig, peer.NewRequest(data), opt)
		cancel()
		if e != nil || rsp.Error() != nil || len(rsp.Data()) > securityv1.MaxMessageSize {
			continue
		}
		var signed securityv1.SignedTLSConfig
		if json.Unmarshal(rsp.Data(), &signed) != nil {
			continue
		}
		if _, e = securityv1.VerifyTLSConfig(input, signed, keys); e != nil {
			continue
		}
		return signed, nil
	}
	return result, errors.New("no trusted TLS provider response")
}

// TLSConfigChanged coalesces provider registration/reconnection notifications.
func (ex *exchange) TLSConfigChanged() {
	select {
	case ex.tlsWake <- struct{}{}:
	default:
	}
}
func (ex *exchange) TLSConfigWake() <-chan struct{} { return ex.tlsWake }
