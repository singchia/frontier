package repo

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/stretchr/testify/require"
)

func TestServiceLease(t *testing.T) {
	s := miniredis.RunT(t)
	now := time.Unix(1700000000, 0)
	s.SetTime(now)
	conf := &config.Configuration{}
	conf.Redis.Mode = "standalone"
	conf.Redis.Standalone.Network = "tcp"
	conf.Redis.Standalone.Addr = s.Addr()
	conf.FrontierManager.Expiration.ServiceMeta = 60
	dao, err := NewDao(conf)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, dao.Close()) })
	ctx := context.Background()
	a := &Service{FrontierID: "a", Service: "app", Addr: "127.0.0.1:1001"}
	b := &Service{FrontierID: "b", Service: "app", Addr: "127.0.0.1:1002"}
	advance := func(d time.Duration) { now = now.Add(d); s.SetTime(now); s.FastForward(d) }
	require.NoError(t, dao.SetServiceAndAlive(1, a, 30*time.Second))
	require.NoError(t, dao.SetServiceAndAlive(1, a, 30*time.Second))
	require.Equal(t, "1", s.HGet(getFrontierKey("a"), "service_count"))
	require.NoError(t, dao.SetServiceAndAlive(1, b, 30*time.Second))
	got, err := dao.GetServices([]uint64{1, 99})
	require.NoError(t, err)
	require.Equal(t, []*Service{a, b}, got)
	advance(20 * time.Second)
	require.NoError(t, dao.HeartbeatService(ctx, 1, a, 30*time.Second))
	advance(11 * time.Second)
	// a 的心跳不能延长 b 的在线时间，查询必须触发清理而不是等待下次心跳。
	got, err = dao.GetServices([]uint64{1})
	require.NoError(t, err)
	require.Equal(t, []*Service{a}, got)
	require.Equal(t, "", s.HGet(getServiceKey(1), "b"))
	require.True(t, s.Exists(getAliveServiceKey(1)))
	ids, err := dao.GetAllServiceIDs()
	require.NoError(t, err)
	require.Equal(t, []uint64{1}, ids)
	page, _, err := dao.GetServicesByCursor(&ServiceQuery{Count: 100})
	require.NoError(t, err)
	require.Equal(t, []*Service{a}, page)
	advance(31 * time.Second)
	_, err = dao.GetService(1)
	require.ErrorIs(t, err, redis.Nil)
	require.False(t, s.Exists(getAliveServiceKey(1)))
	// 完整心跳恢复过期记录，并发重试不放大计数。
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- dao.HeartbeatService(ctx, 1, a, 30*time.Second) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, "1", s.HGet(getFrontierKey("a"), "service_count"))
	require.NoError(t, dao.HeartbeatService(ctx, 1, b, 30*time.Second))
	require.NoError(t, dao.DeleteServiceInstance(ctx, 1, "a", a.Addr))
	got, err = dao.GetServices([]uint64{1})
	require.NoError(t, err)
	require.Equal(t, []*Service{b}, got)
	require.True(t, s.Exists(getAliveServiceKey(1)))
	require.NoError(t, dao.DeleteServiceInstance(ctx, 1, "a", a.Addr))
	require.Equal(t, "0", s.HGet(getFrontierKey("a"), "service_count"))
	// 同一个 Frontier 的新连接替换后，旧心跳与旧 Offline 均不能修改新记录。
	newer := &Service{FrontierID: "b", Service: "app", Addr: "127.0.0.1:2002"}
	require.NoError(t, dao.SetServiceAndAlive(1, newer, 30*time.Second))
	advance(time.Second)
	require.ErrorIs(t, dao.HeartbeatService(ctx, 1, b, 30*time.Second), ErrServiceOwnerChanged)
	require.ErrorIs(t, dao.DeleteServiceInstance(ctx, 1, "b", b.Addr), ErrServiceOwnerChanged)
	current, err := dao.GetService(1)
	require.NoError(t, err)
	require.Equal(t, newer, current)
	require.Equal(t, 29*time.Second, s.TTL(getAliveServiceKey(1)))
	// 旧格式仅允许已有归属续期，不可为缺失映射造空记录。
	require.NoError(t, dao.HeartbeatService(ctx, 1, &Service{FrontierID: "b"}, 30*time.Second))
	require.ErrorIs(t, dao.HeartbeatService(ctx, 2, &Service{FrontierID: "b"}, 30*time.Second), ErrServiceRegistrationMissing)
	require.Error(t, dao.ExpireService(1, 30*time.Second))
	// 旧版 Hash 值在新 Frontier 完整心跳后转换，保持已有字段协议可读。
	s.HSet(getServiceKey(3), "a", "service: app, frontier_id: a, addr: 127.0.0.1:1001, update_time: 1")
	require.NoError(t, dao.HeartbeatService(ctx, 3, a, 30*time.Second))
	require.True(t, json.Valid([]byte(s.HGet(getServiceKey(3), "a"))))
	s.SetError("ERR injected")
	require.Error(t, dao.HeartbeatService(ctx, 1, b, 30*time.Second))
	s.SetError("")
}
