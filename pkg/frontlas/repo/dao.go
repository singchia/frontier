package repo

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/singchia/frontier/pkg/frontlas/apis"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"k8s.io/klog/v2"
)

const (
	modeStandalone = iota
	modeSentinel
	modeCluster
)

type RDS interface {
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	Get(ctx context.Context, key string) *redis.StringCmd
	MGet(ctx context.Context, keys ...string) *redis.SliceCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Keys(ctx context.Context, pattern string) *redis.StringSliceCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd

	TxPipeline() redis.Pipeliner
	Close() error
}

type Dao struct {
	mode int
	rds  RDS
}

func newDao(config *config.Configuration) (*Dao, error) {
	var (
		rds  RDS
		mode int
	)
	conf := config.Redis
	switch conf.Mode {
	case "standalone":
		sconf := conf.Standalone
		opt := &redis.Options{
			Network:          sconf.Network,
			Addr:             sconf.Addr,
			ClientName:       sconf.ClientName,
			Protocol:         sconf.Protocol,
			Username:         sconf.Username,
			Password:         sconf.Password,
			DB:               sconf.DB,
			MaxRetries:       sconf.MaxRetries,
			MinRetryBackoff:  time.Duration(sconf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:  time.Duration(sconf.MaxRetryBackoff) * time.Second,
			DialTimeout:      time.Duration(sconf.DialTimeout) * time.Second,
			ReadTimeout:      time.Duration(sconf.ReadTimeout) * time.Second,
			WriteTimeout:     time.Duration(sconf.WriteTimeout) * time.Second,
			PoolFIFO:         sconf.PoolFIFO,
			PoolSize:         sconf.PoolSize,
			PoolTimeout:      time.Duration(sconf.PoolTimeout) * time.Second,
			MinIdleConns:     sconf.MinIdleConns,
			MaxIdleConns:     sconf.MaxIdleConns,
			ConnMaxIdleTime:  time.Duration(sconf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:  time.Duration(sconf.ConnMaxLifetime) * time.Second,
			DisableIndentity: sconf.DisableIndentity,
			IdentitySuffix:   sconf.IdentitySuffix,
		}
		rds = redis.NewClient(opt)
		_, err := rds.Ping(context.TODO()).Result()
		if err != nil {
			klog.Errorf("redis standalone ping err: %s", err)
			return nil, err
		}
		mode = modeStandalone

	case "sentinel":
		sconf := conf.Sentinel
		opt := &redis.FailoverOptions{
			MasterName:              sconf.MasterName,
			SentinelAddrs:           sconf.Addrs,
			Protocol:                sconf.Protocol,
			Username:                sconf.Username,
			Password:                sconf.Password,
			DB:                      sconf.DB,
			ClientName:              sconf.ClientName,
			RouteByLatency:          sconf.RouteByLatency,
			RouteRandomly:           sconf.RouteRandomly,
			ReplicaOnly:             sconf.ReplicaOnly,
			UseDisconnectedReplicas: sconf.UseDisconnectedReplicas,
			MinRetryBackoff:         time.Duration(sconf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:         time.Duration(sconf.MaxRetryBackoff) * time.Second,
			DialTimeout:             time.Duration(sconf.DialTimeout) * time.Second,
			ReadTimeout:             time.Duration(sconf.ReadTimeout) * time.Second,
			WriteTimeout:            time.Duration(sconf.WriteTimeout) * time.Second,
			PoolFIFO:                sconf.PoolFIFO,
			PoolSize:                sconf.PoolSize,
			PoolTimeout:             time.Duration(sconf.PoolTimeout) * time.Second,
			MinIdleConns:            sconf.MinIdleConns,
			MaxIdleConns:            sconf.MaxIdleConns,
			ConnMaxIdleTime:         time.Duration(sconf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:         time.Duration(sconf.ConnMaxLifetime) * time.Second,
			DisableIndentity:        sconf.DisableIndentity,
			IdentitySuffix:          sconf.IdentitySuffix,
		}
		rds = redis.NewFailoverClient(opt)
		_, err := rds.Ping(context.TODO()).Result()
		if err != nil {
			klog.Errorf("redis sentinel ping err: %s", err)
			return nil, err
		}
		mode = modeSentinel

	case "cluster":
		cconf := conf.Cluster
		opt := &redis.ClusterOptions{
			Addrs:            cconf.Addrs,
			Protocol:         cconf.Protocol,
			Username:         cconf.Username,
			Password:         cconf.Password,
			ClientName:       cconf.ClientName,
			MaxRedirects:     cconf.MaxRedirects,
			RouteByLatency:   cconf.RouteByLatency,
			RouteRandomly:    cconf.RouteRandomly,
			MinRetryBackoff:  time.Duration(cconf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:  time.Duration(cconf.MaxRetryBackoff) * time.Second,
			DialTimeout:      time.Duration(cconf.DialTimeout) * time.Second,
			ReadTimeout:      time.Duration(cconf.ReadTimeout) * time.Second,
			WriteTimeout:     time.Duration(cconf.WriteTimeout) * time.Second,
			PoolFIFO:         cconf.PoolFIFO,
			PoolSize:         cconf.PoolSize,
			PoolTimeout:      time.Duration(cconf.PoolTimeout) * time.Second,
			MinIdleConns:     cconf.MinIdleConns,
			MaxIdleConns:     cconf.MaxIdleConns,
			ConnMaxIdleTime:  time.Duration(cconf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:  time.Duration(cconf.ConnMaxLifetime) * time.Second,
			DisableIndentity: cconf.DisableIndentity,
			IdentitySuffix:   cconf.IdentitySuffix,
		}
		rds = redis.NewClusterClient(opt)
		_, err := rds.Ping(context.TODO()).Result()
		if err != nil {
			klog.Errorf("redis cluster ping err: %s", err)
			return nil, err
		}
		mode = modeCluster

	default:
		return nil, apis.ErrUnsupportRedisServerMode
	}
	return &Dao{
		rds:  rds,
		mode: mode,
	}, nil
}

func (dao *Dao) Close() error {
	return dao.rds.Close()
}
