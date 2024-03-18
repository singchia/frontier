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

type Dao struct {
	mode       int
	rds        *redis.Client
	clusterrds *redis.ClusterClient
}

func newDao(config *config.Configuration) (*Dao, error) {
	var (
		rds        *redis.Client
		clusterrds *redis.ClusterClient
		mode       int
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
		clusterrds = redis.NewClusterClient(opt)
		_, err := clusterrds.Ping(context.TODO()).Result()
		if err != nil {
			klog.Errorf("redis cluster ping err: %s", err)
			return nil, err
		}
		mode = modeCluster

	default:
		return nil, apis.ErrUnsupportRedisServerMode
	}
	return &Dao{
		rds:        rds,
		clusterrds: clusterrds,
		mode:       mode,
	}, nil
}

func (dao *Dao) Close() error {
	switch dao.mode {
	case modeStandalone, modeSentinel:
		return dao.rds.Close()

	case modeCluster:
		return dao.clusterrds.Close()
	}
	return nil
}
