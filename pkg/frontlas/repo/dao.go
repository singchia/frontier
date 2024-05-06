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
	Exists(ctx context.Context, keys ...string) *redis.IntCmd

	TxPipeline() redis.Pipeliner
	Close() error
}

type Dao struct {
	mode int
	rds  RDS
}

func NewDao(config *config.Configuration) (*Dao, error) {
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
			ClientName:       conf.ClientName,
			Protocol:         conf.Protocol,
			Username:         conf.Username,
			Password:         conf.Password,
			DB:               sconf.DB,
			MaxRetries:       conf.MaxRetries,
			MinRetryBackoff:  time.Duration(conf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:  time.Duration(conf.MaxRetryBackoff) * time.Second,
			DialTimeout:      time.Duration(conf.DialTimeout) * time.Second,
			ReadTimeout:      time.Duration(conf.ReadTimeout) * time.Second,
			WriteTimeout:     time.Duration(conf.WriteTimeout) * time.Second,
			PoolFIFO:         conf.PoolFIFO,
			PoolSize:         conf.PoolSize,
			PoolTimeout:      time.Duration(conf.PoolTimeout) * time.Second,
			MinIdleConns:     conf.MinIdleConns,
			MaxIdleConns:     conf.MaxIdleConns,
			ConnMaxIdleTime:  time.Duration(conf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:  time.Duration(conf.ConnMaxLifetime) * time.Second,
			DisableIndentity: conf.DisableIndentity,
			IdentitySuffix:   conf.IdentitySuffix,
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
			Protocol:                conf.Protocol,
			Username:                conf.Username,
			Password:                conf.Password,
			DB:                      sconf.DB,
			ClientName:              conf.ClientName,
			RouteByLatency:          sconf.RouteByLatency,
			RouteRandomly:           sconf.RouteRandomly,
			ReplicaOnly:             sconf.ReplicaOnly,
			UseDisconnectedReplicas: sconf.UseDisconnectedReplicas,
			MinRetryBackoff:         time.Duration(conf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:         time.Duration(conf.MaxRetryBackoff) * time.Second,
			DialTimeout:             time.Duration(conf.DialTimeout) * time.Second,
			ReadTimeout:             time.Duration(conf.ReadTimeout) * time.Second,
			WriteTimeout:            time.Duration(conf.WriteTimeout) * time.Second,
			PoolFIFO:                conf.PoolFIFO,
			PoolSize:                conf.PoolSize,
			PoolTimeout:             time.Duration(conf.PoolTimeout) * time.Second,
			MinIdleConns:            conf.MinIdleConns,
			MaxIdleConns:            conf.MaxIdleConns,
			ConnMaxIdleTime:         time.Duration(conf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:         time.Duration(conf.ConnMaxLifetime) * time.Second,
			DisableIndentity:        conf.DisableIndentity,
			IdentitySuffix:          conf.IdentitySuffix,
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
			Protocol:         conf.Protocol,
			Username:         conf.Username,
			Password:         conf.Password,
			ClientName:       conf.ClientName,
			MaxRedirects:     cconf.MaxRedirects,
			RouteByLatency:   cconf.RouteByLatency,
			RouteRandomly:    cconf.RouteRandomly,
			MinRetryBackoff:  time.Duration(conf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:  time.Duration(conf.MaxRetryBackoff) * time.Second,
			DialTimeout:      time.Duration(conf.DialTimeout) * time.Second,
			ReadTimeout:      time.Duration(conf.ReadTimeout) * time.Second,
			WriteTimeout:     time.Duration(conf.WriteTimeout) * time.Second,
			PoolFIFO:         conf.PoolFIFO,
			PoolSize:         conf.PoolSize,
			PoolTimeout:      time.Duration(conf.PoolTimeout) * time.Second,
			MinIdleConns:     conf.MinIdleConns,
			MaxIdleConns:     conf.MaxIdleConns,
			ConnMaxIdleTime:  time.Duration(conf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:  time.Duration(conf.ConnMaxLifetime) * time.Second,
			DisableIndentity: conf.DisableIndentity,
			IdentitySuffix:   conf.IdentitySuffix,
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
