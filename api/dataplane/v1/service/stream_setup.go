package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/geminio"
)

// ErrEdgeNotOnline means the contacted Frontier has no local session for the Edge.
// It is distinct from an application connection failure or a transport timeout.
var ErrEdgeNotOnline = apis.ErrEdgeNotOnline

// ErrEdgeStreamOpen means the Edge exists but its next-hop stream could not open.
var ErrEdgeStreamOpen = errors.New("frontier could not open edge stream")

// streamSetupError is scoped to one stream and must not evict its shared transport.
type streamSetupError struct{ err error }

func (e *streamSetupError) Error() string { return e.err.Error() }
func (e *streamSetupError) Unwrap() error { return e.err }

func confirmEdgeStream(ctx context.Context, st geminio.Stream) (geminio.Stream, error) {
	rsp, err := st.Call(ctx, apis.RPCStreamSetup, st.NewRequest(nil))
	// A legacy Frontier forwards this reserved RPC to the Edge. Old Geminio
	// returns this exact unknown-method error without consuming any raw bytes.
	// EOF, timeout and arbitrary application errors are NOT compatibility signals.
	if err != nil && err.Error() == "no such rpc: "+apis.RPCStreamSetup && ctx.Err() == nil {
		return st, nil
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	if err == nil {
		switch string(rsp.Data()) {
		case apis.StreamReady:
			return st, nil
		case apis.StreamEdgeOffline:
			err = ErrEdgeNotOnline
		case apis.StreamEdgeOpenFailed:
			err = ErrEdgeStreamOpen
		default:
			err = errors.New("invalid frontier stream setup response")
		}
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	// Setup failed before the caller obtained a stream; discard without graceful drain.
	_ = geminio.AbortStream(st)
	return nil, &streamSetupError{err: fmt.Errorf("frontier stream setup: %w", err)}
}
