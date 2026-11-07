package e2e

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/stretchr/testify/require"
)

func TestResetCrossesFrontierPair(t *testing.T) {
	svc := newService(t, service.OptionServiceName("reset-pair"))
	ed := newEdge(t)
	baseline := len(ed.ListStreams())
	for round := 0; round < 8; round++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		a, err := svc.OpenStream(ctx, ed.EdgeID())
		cancel()
		require.NoError(t, err)
		b, err := ed.AcceptStream()
		require.NoError(t, err)
		// Service does not read. Saturate both Gemini hops through real forwarding.
		require.NoError(t, b.SetWriteDeadline(time.Now().Add(150*time.Millisecond)))
		buf := make([]byte, 32768)
		for {
			_, err = b.Write(buf)
			if err != nil {
				break
			}
		}
		require.True(t, errors.Is(err, os.ErrDeadlineExceeded), "got %v", err)
		require.NoError(t, geminio.AbortStream(a))
		require.NoError(t, b.SetReadDeadline(time.Now().Add(3*time.Second)))
		_, err = b.Read(make([]byte, 1))
		require.ErrorIs(t, err, geminio.ErrStreamAborted)
		require.Eventually(t, func() bool { return len(ed.ListStreams()) == baseline }, 3*time.Second, time.Millisecond)
	}
	// A fresh flow still traverses the same service connection.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	a, err := svc.OpenStream(ctx, ed.EdgeID())
	require.NoError(t, err)
	b, err := ed.AcceptStream()
	require.NoError(t, err)
	require.NoError(t, geminio.AbortStream(a))
	b.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, err = b.Read(make([]byte, 1))
	require.ErrorIs(t, err, geminio.ErrStreamAborted)
}
