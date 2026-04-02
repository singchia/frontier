package security

import (
	"context"
	"testing"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/require"
)

// SEC-FUZZ-001: Random bytes as Edge meta must not crash frontier
func FuzzEdgeMeta(f *testing.F) {
	// seed corpus
	f.Add([]byte("normal-meta"))
	f.Add([]byte{0x00})
	f.Add([]byte{0xff, 0xfe, 0xfd})
	f.Add([]byte("line1\nline2"))

	f.Fuzz(func(t *testing.T, meta []byte) {
		e, err := edge.NewEdge(testEdgeDial, edge.OptionEdgeMeta(meta))
		if err != nil {
			return // connection refused or rejected is acceptable
		}
		e.Close()
	})
}

// SEC-FUZZ-002: Random bytes as RPC payload must not crash frontier
func FuzzRPCPayload(f *testing.F) {
	f.Add([]byte("hello"))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0xff})

	// set up a long-lived service that echoes RPCs
	svc, err := service.NewService(testSvcDial, service.OptionServiceName("fuzz-rpc-svc"))
	require.NoError(f, err)
	f.Cleanup(func() { svc.Close() })
	require.NoError(f, svc.Register(context.TODO(), "fuzz", func(_ context.Context, req geminio.Request, resp geminio.Response) {
		resp.SetData(req.Data())
	}))

	f.Fuzz(func(t *testing.T, payload []byte) {
		e, err := edge.NewEdge(testEdgeDial)
		if err != nil {
			return
		}
		defer e.Close()
		req := e.NewRequest(payload)
		_, _ = e.Call(context.TODO(), "fuzz", req)
	})
}

// SEC-FUZZ-003: Random bytes as Publish payload must not crash frontier
func FuzzMessagePayload(f *testing.F) {
	f.Add([]byte("msg"))
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x02})

	const topic = "fuzz-topic"
	svc, err := service.NewService(testSvcDial,
		service.OptionServiceName("fuzz-msg-svc"),
		service.OptionServiceReceiveTopics([]string{topic}),
	)
	require.NoError(f, err)
	f.Cleanup(func() { svc.Close() })
	// drain received messages silently
	go func() {
		for {
			msg, err := svc.Receive(context.TODO())
			if err != nil {
				return
			}
			msg.Done()
		}
	}()

	f.Fuzz(func(t *testing.T, payload []byte) {
		e, err := edge.NewEdge(testEdgeDial)
		if err != nil {
			return
		}
		defer e.Close()
		msg := e.NewMessage(payload)
		_ = e.Publish(context.TODO(), topic, msg)
	})
}
