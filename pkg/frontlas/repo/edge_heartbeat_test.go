package repo

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/stretchr/testify/require"
)

func TestHeartbeatEdgeRecovery(t *testing.T) {
	s := miniredis.RunT(t)
	conf := &config.Configuration{}
	conf.Redis.Mode = "standalone"
	conf.Redis.Standalone.Network = "tcp"
	conf.Redis.Standalone.Addr = s.Addr()
	conf.FrontierManager.Expiration.EdgeMeta = 60
	conf.FrontierManager.Expiration.ServiceMeta = 120
	dao, err := NewDao(conf)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, dao.Close()) })
	ctx := context.Background()
	edge := &Edge{FrontierID: "a", Addr: "127.0.0.1:1001", UpdateTime: 1}
	require.NoError(t, dao.SetEdgeAndAlive(1, edge, 30*time.Second))
	require.Equal(t, 60*time.Second, s.TTL(getEdgeKey(1)), "must use EdgeMeta, not ServiceMeta")

	t.Run("expired registration recreated concurrently without inflating counts", func(t *testing.T) {
		s.FastForward(61 * time.Second)
		require.False(t, s.Exists(getEdgeKey(1)))
		var wg sync.WaitGroup
		errs := make(chan error, 20)
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); errs <- dao.HeartbeatEdge(ctx, 1, edge, 30*time.Second) }()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		got, err := dao.GetEdge(1)
		require.NoError(t, err)
		require.Equal(t, edge, got)
		require.True(t, s.Exists(getAliveEdgeKey(1)))
		require.Equal(t, "1", s.HGet(getFrontierKey("a"), "edge_count"))
	})
	t.Run("missing alive marker restored", func(t *testing.T) {
		s.FastForward(31 * time.Second)
		require.False(t, s.Exists(getAliveEdgeKey(1)))
		require.NoError(t, dao.HeartbeatEdge(ctx, 1, edge, 30*time.Second))
		require.Equal(t, 30*time.Second, s.TTL(getAliveEdgeKey(1)))
		require.Equal(t, 60*time.Second, s.TTL(getEdgeKey(1)))
	})
	t.Run("legacy heartbeat only renews existing registration", func(t *testing.T) {
		legacy := &Edge{FrontierID: "a"}
		require.NoError(t, dao.HeartbeatEdge(ctx, 1, legacy, 30*time.Second))
		require.ErrorIs(t, dao.HeartbeatEdge(ctx, 2, legacy, 30*time.Second), ErrEdgeRegistrationMissing)
		require.False(t, s.Exists(getAliveEdgeKey(2)))
	})
	t.Run("old heartbeat cannot overwrite or renew new owner", func(t *testing.T) {
		for _, next := range []*Edge{{FrontierID: "b", Addr: "127.0.0.1:2001"}, {FrontierID: "a", Addr: "127.0.0.1:3001"}} {
			require.NoError(t, dao.SetEdgeAndAlive(1, next, 30*time.Second))
			s.FastForward(time.Second)
			require.ErrorIs(t, dao.HeartbeatEdge(ctx, 1, edge, 30*time.Second), ErrEdgeOwnerChanged)
			got, err := dao.GetEdge(1)
			require.NoError(t, err)
			require.Equal(t, next, got)
			require.Equal(t, 29*time.Second, s.TTL(getAliveEdgeKey(1)))
		}
	})
	t.Run("redis errors propagated", func(t *testing.T) {
		s.SetError("ERR injected")
		require.Error(t, dao.HeartbeatEdge(ctx, 1, edge, 30*time.Second))
		s.SetError("")
	})
}
