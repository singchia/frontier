//go:build integration

package repo

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/stretchr/testify/require"
)

// 专用 Redis 实例由测试运行方提供，不清库，只删除本测试生成的键。
func TestServiceLeaseRealRedis(t *testing.T) {
	addr := os.Getenv("FRONTLAS_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set FRONTLAS_TEST_REDIS_ADDR to an isolated Redis")
	}
	conf := &config.Configuration{}
	conf.Redis.Mode = "standalone"
	conf.Redis.Standalone.Network = "tcp"
	conf.Redis.Standalone.Addr = addr
	conf.FrontierManager.Expiration.ServiceMeta = 60
	dao, err := NewDao(conf)
	require.NoError(t, err)
	ctx := context.Background()
	id := uint64(time.Now().UnixNano())
	prefix := "lease-test-" + strconv.FormatUint(id, 10)
	a := &Service{FrontierID: prefix + "a", Service: "app", Addr: "127.0.0.1:1001"}
	b := &Service{FrontierID: prefix + "b", Service: "app", Addr: "127.0.0.1:1002"}
	t.Cleanup(func() {
		require.NoError(t, dao.rds.Del(ctx, getServiceKey(id), getServiceLeasesKey(id), getAliveServiceKey(id), getFrontierKey(a.FrontierID), getFrontierKey(b.FrontierID)).Err())
		require.NoError(t, dao.Close())
	})
	require.NoError(t, dao.SetServiceAndAlive(id, a, 2*time.Second))
	require.NoError(t, dao.SetServiceAndAlive(id, b, 300*time.Millisecond))
	// a 的有效租约足够长；让 b 自然到期，不能靠模拟时钟掩盖 Lua TIME 差异。
	time.Sleep(400 * time.Millisecond)
	require.NoError(t, dao.HeartbeatService(ctx, id, a, 2*time.Second))
	instances, err := dao.GetServices([]uint64{id})
	require.NoError(t, err)
	require.Equal(t, []*Service{a}, instances)
	// 模拟注册数据在故障中丢失，保持 Frontier 上的实例不变。
	require.NoError(t, dao.rds.Del(ctx, getServiceKey(id), getServiceLeasesKey(id), getAliveServiceKey(id)).Err())
	_, err = dao.GetService(id)
	require.ErrorIs(t, err, redis.Nil)
	require.NoError(t, dao.HeartbeatService(ctx, id, a, 2*time.Second))
	current, err := dao.GetService(id)
	require.NoError(t, err)
	require.Equal(t, a, current)
	newer := &Service{FrontierID: a.FrontierID, Service: "app", Addr: "127.0.0.1:2001"}
	require.NoError(t, dao.SetServiceAndAlive(id, newer, 2*time.Second))
	require.ErrorIs(t, dao.HeartbeatService(ctx, id, a, 2*time.Second), ErrServiceOwnerChanged)
	require.ErrorIs(t, dao.DeleteServiceInstance(ctx, id, a.FrontierID, a.Addr), ErrServiceOwnerChanged)
	current, err = dao.GetService(id)
	require.NoError(t, err)
	require.Equal(t, newer, current)
}
