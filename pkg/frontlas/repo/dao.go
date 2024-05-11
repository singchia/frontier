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
	conf *config.Configuration
	mode int
	rds  RDS
}

func NewDao(conf *config.Configuration) (*Dao, error) {
	var (
		rds  RDS
		mode int
	)
	redisconf := conf.Redis
	switch redisconf.Mode {
	case "standalone":
		sconf := redisconf.Standalone
		opt := &redis.Options{
			Network:          sconf.Network,
			Addr:             sconf.Addr,
			ClientName:       redisconf.ClientName,
			Protocol:         redisconf.Protocol,
			Username:         redisconf.Username,
			Password:         redisconf.Password,
			DB:               sconf.DB,
			MaxRetries:       redisconf.MaxRetries,
			MinRetryBackoff:  time.Duration(redisconf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:  time.Duration(redisconf.MaxRetryBackoff) * time.Second,
			DialTimeout:      time.Duration(redisconf.DialTimeout) * time.Second,
			ReadTimeout:      time.Duration(redisconf.ReadTimeout) * time.Second,
			WriteTimeout:     time.Duration(redisconf.WriteTimeout) * time.Second,
			PoolFIFO:         redisconf.PoolFIFO,
			PoolSize:         redisconf.PoolSize,
			PoolTimeout:      time.Duration(redisconf.PoolTimeout) * time.Second,
			MinIdleConns:     redisconf.MinIdleConns,
			MaxIdleConns:     redisconf.MaxIdleConns,
			ConnMaxIdleTime:  time.Duration(redisconf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:  time.Duration(redisconf.ConnMaxLifetime) * time.Second,
			DisableIndentity: redisconf.DisableIndentity,
			IdentitySuffix:   redisconf.IdentitySuffix,
		}
		rds = redis.NewClient(opt)
		_, err := rds.Ping(context.TODO()).Result()
		if err != nil {
			klog.Errorf("redis standalone ping err: %s", err)
			return nil, err
		}
		mode = modeStandalone

	case "sentinel":
		sconf := redisconf.Sentinel
		opt := &redis.FailoverOptions{
			MasterName:              sconf.MasterName,
			SentinelAddrs:           sconf.Addrs,
			Protocol:                redisconf.Protocol,
			Username:                redisconf.Username,
			Password:                redisconf.Password,
			DB:                      sconf.DB,
			ClientName:              redisconf.ClientName,
			RouteByLatency:          sconf.RouteByLatency,
			RouteRandomly:           sconf.RouteRandomly,
			ReplicaOnly:             sconf.ReplicaOnly,
			UseDisconnectedReplicas: sconf.UseDisconnectedReplicas,
			MinRetryBackoff:         time.Duration(redisconf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:         time.Duration(redisconf.MaxRetryBackoff) * time.Second,
			DialTimeout:             time.Duration(redisconf.DialTimeout) * time.Second,
			ReadTimeout:             time.Duration(redisconf.ReadTimeout) * time.Second,
			WriteTimeout:            time.Duration(redisconf.WriteTimeout) * time.Second,
			PoolFIFO:                redisconf.PoolFIFO,
			PoolSize:                redisconf.PoolSize,
			PoolTimeout:             time.Duration(redisconf.PoolTimeout) * time.Second,
			MinIdleConns:            redisconf.MinIdleConns,
			MaxIdleConns:            redisconf.MaxIdleConns,
			ConnMaxIdleTime:         time.Duration(redisconf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:         time.Duration(redisconf.ConnMaxLifetime) * time.Second,
			DisableIndentity:        redisconf.DisableIndentity,
			IdentitySuffix:          redisconf.IdentitySuffix,
		}
		rds = redis.NewFailoverClient(opt)
		_, err := rds.Ping(context.TODO()).Result()
		if err != nil {
			klog.Errorf("redis sentinel ping err: %s", err)
			return nil, err
		}
		mode = modeSentinel

	case "cluster":
		cconf := redisconf.Cluster
		opt := &redis.ClusterOptions{
			Addrs:            cconf.Addrs,
			Protocol:         redisconf.Protocol,
			Username:         redisconf.Username,
			Password:         redisconf.Password,
			ClientName:       redisconf.ClientName,
			MaxRedirects:     cconf.MaxRedirects,
			RouteByLatency:   cconf.RouteByLatency,
			RouteRandomly:    cconf.RouteRandomly,
			MinRetryBackoff:  time.Duration(redisconf.MinRetryBackoff) * time.Second,
			MaxRetryBackoff:  time.Duration(redisconf.MaxRetryBackoff) * time.Second,
			DialTimeout:      time.Duration(redisconf.DialTimeout) * time.Second,
			ReadTimeout:      time.Duration(redisconf.ReadTimeout) * time.Second,
			WriteTimeout:     time.Duration(redisconf.WriteTimeout) * time.Second,
			PoolFIFO:         redisconf.PoolFIFO,
			PoolSize:         redisconf.PoolSize,
			PoolTimeout:      time.Duration(redisconf.PoolTimeout) * time.Second,
			MinIdleConns:     redisconf.MinIdleConns,
			MaxIdleConns:     redisconf.MaxIdleConns,
			ConnMaxIdleTime:  time.Duration(redisconf.ConnMaxIdleTime) * time.Second,
			ConnMaxLifetime:  time.Duration(redisconf.ConnMaxLifetime) * time.Second,
			DisableIndentity: redisconf.DisableIndentity,
			IdentitySuffix:   redisconf.IdentitySuffix,
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
		conf: conf,
		rds:  rds,
		mode: mode,
	}, nil
}

func (dao *Dao) Close() error {
	return dao.rds.Close()
}
