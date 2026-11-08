package service

import (
	"context"
	"encoding/json"
	"errors"

	securityv1 "github.com/singchia/frontier/api/dataplane/v1/security"
	"github.com/singchia/geminio"
)

type GetTLSConfig func(context.Context, securityv1.TLSConfigRequest) (securityv1.SignedTLSConfig, error)

// RegisterGetTLSConfig uses existing registration fan-out without extending Service.
// The provider must authorize the requested Frontier/CSR before issuing a certificate.
func RegisterGetTLSConfig(ctx context.Context, svc Service, get GetTLSConfig) error {
	if get == nil {
		return errors.New("TLS configuration callback is required")
	}
	return svc.Register(ctx, securityv1.RPCGetTLSConfig, func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		var input securityv1.TLSConfigRequest
		if len(req.Data()) > 65536 || json.Unmarshal(req.Data(), &input) != nil || input.FrontierID == "" || len(input.Nonce) != 64 || len(input.CSR) == 0 {
			rsp.SetError(errors.New("invalid TLS configuration request"))
			return
		}
		result, err := get(ctx, input)
		if err != nil {
			rsp.SetError(errors.New("TLS configuration unavailable"))
			return
		}
		b, err := json.Marshal(result)
		if err != nil || len(b) > securityv1.MaxMessageSize {
			rsp.SetError(errors.New("invalid TLS configuration response"))
			return
		}
		rsp.SetData(b)
	})
}
