package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// E2E-STRM-001: Edge opens a stream to Service, Service accepts it
func TestEdgeOpenStreamToService(t *testing.T) {
	accepted := make(chan geminio.Stream, 1)
	svc := newService(t, service.OptionServiceName("stream-service"))
	go func() {
		st, err := svc.AcceptStream()
		if err == nil {
			accepted <- st
		}
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	st, err := e.OpenStream("stream-service")
	require.NoError(t, err)
	t.Cleanup(func() { st.Close() })

	select {
	case serverSt := <-accepted:
		assert.NotNil(t, serverSt)
		serverSt.Close()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for AcceptStream")
	}
}

// E2E-STRM-002: Service opens a stream to Edge, Edge accepts it
func TestServiceOpenStreamToEdge(t *testing.T) {
	accepted := make(chan geminio.Stream, 1)
	e := newEdge(t)
	go func() {
		st, err := e.AcceptStream()
		if err == nil {
			accepted <- st
		}
	}()

	time.Sleep(30 * time.Millisecond)

	svc := newService(t, service.OptionServiceName("stream-opener"))
	st, err := svc.OpenStream(context.TODO(), e.EdgeID())
	require.NoError(t, err)
	t.Cleanup(func() { st.Close() })

	select {
	case edgeSt := <-accepted:
		assert.NotNil(t, edgeSt)
		edgeSt.Close()
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for AcceptStream on edge")
	}
}

// E2E-STRM-003: Raw IO forwarded bidirectionally through the stream
func TestStreamRawDataForward(t *testing.T) {
	serverRead := make(chan []byte, 1)
	clientRead := make(chan []byte, 1)

	svc := newService(t, service.OptionServiceName("raw-echo"))
	go func() {
		st, err := svc.AcceptStream()
		if err != nil {
			return
		}
		defer st.Close()
		buf := make([]byte, 64)
		n, _ := st.Read(buf)
		serverRead <- buf[:n]
		st.Write([]byte("server-reply"))
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	st, err := e.OpenStream("raw-echo")
	require.NoError(t, err)
	defer st.Close()

	_, err = st.Write([]byte("client-hello"))
	require.NoError(t, err)

	buf := make([]byte, 64)
	go func() {
		n, _ := st.Read(buf)
		clientRead <- buf[:n]
	}()

	select {
	case data := <-serverRead:
		assert.Equal(t, []byte("client-hello"), data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server read")
	}
	select {
	case data := <-clientRead:
		assert.Equal(t, []byte("server-reply"), data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for client read")
	}
}

// E2E-STRM-004: Message forwarded bidirectionally inside a stream
func TestStreamMessageForward(t *testing.T) {
	const streamTopic = "stream-topic"
	serverReceived := make(chan []byte, 1)
	clientReceived := make(chan []byte, 1)

	svc := newService(t, service.OptionServiceName("msg-echo"))
	go func() {
		st, err := svc.AcceptStream()
		if err != nil {
			return
		}
		defer st.Close()
		// receive from edge
		msg, err := st.Receive(context.TODO())
		if err != nil {
			return
		}
		serverReceived <- msg.Data()
		msg.Done()
		// reply back
		reply := st.NewMessage([]byte("svc-msg-reply"))
		_ = st.Publish(context.TODO(), reply)
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	st, err := e.OpenStream("msg-echo")
	require.NoError(t, err)
	defer st.Close()

	go func() {
		msg, err := st.Receive(context.TODO())
		if err == nil {
			clientReceived <- msg.Data()
			msg.Done()
		}
	}()

	edgeMsg := st.NewMessage([]byte("edge-msg"))
	err = st.Publish(context.TODO(), edgeMsg)
	require.NoError(t, err)

	select {
	case data := <-serverReceived:
		assert.Equal(t, []byte("edge-msg"), data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for server message")
	}
	select {
	case data := <-clientReceived:
		assert.Equal(t, []byte("svc-msg-reply"), data)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for client message")
	}
}

// E2E-STRM-005: RPC forwarded bidirectionally inside a stream
func TestStreamRPCForward(t *testing.T) {
	svc := newService(t, service.OptionServiceName("rpc-echo"))
	go func() {
		st, err := svc.AcceptStream()
		if err != nil {
			return
		}
		defer st.Close()
		_ = st.Register(context.TODO(), "echo", func(_ context.Context, req geminio.Request, resp geminio.Response) {
			resp.SetData(req.Data())
		})
		// keep the stream alive while the test runs
		time.Sleep(3 * time.Second)
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	st, err := e.OpenStream("rpc-echo")
	require.NoError(t, err)
	defer st.Close()

	time.Sleep(30 * time.Millisecond)

	payload := []byte("rpc-payload")
	req := st.NewRequest(payload)
	resp, err := st.Call(context.TODO(), "echo", req)
	require.NoError(t, err)
	assert.Equal(t, payload, resp.Data())
}

// E2E-STRM-006: Stream Close does not panic and can be called multiple times safely.
func TestStreamClose(t *testing.T) {
	svc := newService(t, service.OptionServiceName("close-test"))
	go func() {
		for {
			st, err := svc.AcceptStream()
			if err != nil {
				return
			}
			st.Close()
		}
	}()

	time.Sleep(30 * time.Millisecond)

	e := newEdge(t)
	st, err := e.OpenStream("close-test")
	require.NoError(t, err)

	// Close must not panic, even when called multiple times
	assert.NotPanics(t, func() { st.Close() })
	assert.NotPanics(t, func() { st.Close() })
}

// E2E-STRM-007: Service opens a stream to an offline edge; the stream is returned
// but immediately closed by frontier (edge not found), so subsequent IO fails.
func TestStreamTargetEdgeOffline(t *testing.T) {
	offlineEdge, err := edge.NewEdge(edgeDialer())
	require.NoError(t, err)
	offlineID := offlineEdge.EdgeID()
	offlineEdge.Close()

	time.Sleep(50 * time.Millisecond)

	svc := newService(t, service.OptionServiceName("opener"))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	st, err := svc.OpenStream(ctx, offlineID)
	// frontier may return an error immediately, or return a stream that is
	// already closed — either way IO must fail.
	if err != nil {
		return // expected: direct error
	}
	defer st.Close()
	// if a stream was returned, a write or receive must fail
	_, writeErr := st.Write([]byte("probe"))
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer recvCancel()
	_, recvErr := st.Receive(recvCtx)
	if writeErr == nil && recvErr == nil {
		t.Error("expected IO on stream to dead edge to fail, but both succeeded")
	}
}
