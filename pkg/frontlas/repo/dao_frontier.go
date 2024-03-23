package repo

import (
	"context"
	_ "embed"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/singchia/frontier/pkg/frontlas/apis"
	"k8s.io/klog/v2"
)

// we set expire time to indicate real stats of edges
const (
	frontiersKeyPrefix    = "frontlas:frontiers:" // example: frontlas:frontiers:123 "{}"
	frontiersKeyPrefixAll = "frontlas:frontiers:*"

	frontiersAliveKeyPrefix = "frontlas:alive:frontiers:" // example: frontlas:alive:frontiers:123 1 ex 20
)

//go:embed lua/mhgetall.lua
var mhgetallScript string

//go:embed lua/frontier_create.lua
var frontierCreateScript string

func (dao *Dao) GetAllFrontierIDs() ([]string, error) {
	keys, err := dao.rds.Keys(context.TODO(), frontiersKeyPrefixAll).Result()
	if err != nil {
		klog.Errorf("dao get all frontierIDs, keys err: %s", err)
		return nil, err
	}

	frontierIDs := []string{}
	for _, v := range keys {
		frontierID := getFrontierID(v)
		frontierIDs = append(frontierIDs, frontierID)
	}
	return frontierIDs, nil
}

func (dao *Dao) GetAllFrontiers() ([]*Frontier, error) {
	keys, err := dao.rds.Keys(context.TODO(), frontiersKeyPrefixAll).Result()
	if err != nil {
		klog.Errorf("dao get all frontierIDs, keys err: %s", err)
		return nil, err
	}
	if keys == nil || len(keys) == 0 {
		return []*Frontier{}, nil
	}
	return dao.getFrontiers(keys)
}

func (dao *Dao) GetFrontiers(frontierIDs []string) ([]*Frontier, error) {
	keys := make([]string, len(frontierIDs))
	for i, frontierID := range frontierIDs {
		keys[i] = getFrontierID(frontierID)
	}
	return dao.getFrontiers(keys)
}

func (dao *Dao) getFrontiers(keys []string) ([]*Frontier, error) {
	results, err := dao.rds.Eval(context.TODO(), mhgetallScript, keys).Result()
	if err != nil {
		klog.Errorf("dao get all frontiers, eval err: %s", err)
		return nil, err
	}
	// [["edge_count", "1"]["edge_count", "0"]]
	elems, ok := results.([]interface{})
	if !ok {
		return nil, apis.ErrWrongTypeInRedis
	}
	return redisArrayToFrontiers(keys, elems)
}

func (dao *Dao) GetFrontier(frontierID string) (*Frontier, error) {
	result, err := dao.rds.HGetAll(context.TODO(), getFrontierKey(frontierID)).Result()
	if err != nil {
		klog.Errorf("dao get frontier, hgetall err: %s", err)
		return nil, err
	}
	frontier := &Frontier{
		FrontierID:                 frontierID,
		AdvertisedServiceboundAddr: result["advertised_sb_addr"],
		AdvertisedEdgeboundAddr:    result["advertised_eb_addr"],
	}
	edgeCount, ok := result["edge_count"]
	if ok {
		count, err := strconv.Atoi(edgeCount)
		if err != nil {
			return nil, err
		}
		frontier.EdgeCount = count
	}
	serviceCount, ok := result["service_count"]
	if ok {
		count, err := strconv.Atoi(serviceCount)
		if err != nil {
			return nil, err
		}
		frontier.ServiceCount = count
	}
	return frontier, nil
}

// obsoleted
func (dao *Dao) SetFrontier(frontierID string, frontier *Frontier) error {
	_, err := dao.rds.HSet(context.TODO(), getFrontierKey(frontierID),
		"advertised_sb_addr", frontier.AdvertisedServiceboundAddr,
		"advertised_eb_addr", frontier.AdvertisedEdgeboundAddr,
		"edge_count", frontier.EdgeCount,
		"service_count", frontier.ServiceCount).Result()
	if err != nil {
		klog.Errorf("dao set frontier, hset err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) SetFrontierAndAlive(frontierID string, frontier *Frontier, expiration time.Duration) (bool, error) {
	result, err := dao.rds.Eval(context.TODO(), frontierCreateScript,
		[]string{getFrontierKey(frontierID), getAliveFrontierKey(frontierID), "advertised_sb_addr", "advertised_eb_addr", "edge_count", "service_count"},
		expiration.Seconds(), frontier.AdvertisedServiceboundAddr, frontier.AdvertisedEdgeboundAddr, frontier.EdgeCount, frontier.ServiceCount).Result()
	if err != nil {
		klog.Errorf("dao set frontier and alive, eval err: %s", err)
		return false, err
	}
	return result.(int64) == 1, nil
}

func (dao *Dao) ExpireFrontier(frontier string, expiration time.Duration) error {
	ok, err := dao.rds.Expire(context.TODO(), getAliveFrontierKey(frontier), expiration).Result()
	if err != nil {
		return err
	}
	if !ok {
		return apis.ErrExpireFailed
	}
	return nil
}

// obsoleted
func (dao *Dao) SetFrontierEdgeCount(frontierID string, edgeCount int) error {
	_, err := dao.rds.HSet(context.TODO(), getFrontierKey(frontierID),
		"edge_count", edgeCount).Result()
	if err != nil {
		klog.Errorf("dao set frontier edge count, hset err: %s", err)
		return err
	}
	return nil
}

// obsoleted
func (dao *Dao) SetFrontierServiceCount(frontierID string, serviceCount int) error {
	_, err := dao.rds.HSet(context.TODO(), getFrontierKey(frontierID),
		"service_count", serviceCount).Result()
	if err != nil {
		klog.Errorf("dao set frontier service count, hset err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) SetFrontierCount(frontierID string, edgeCount, serviceCount int) error {
	pipeliner := dao.rds.TxPipeline()
	pipeliner.HSet(context.TODO(), getFrontierKey(frontierID),
		"edge_count", edgeCount)
	pipeliner.HSet(context.TODO(), getFrontierKey(frontierID),
		"service_count", serviceCount)
	_, err := pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao set frontier count, pipeliner exec err: %s", err)
		return err
	}
	return nil
}

// we keep frontier but delete alive:frontier
func (dao *Dao) DeleteFrontier(frontier string) error {
	pipeliner := dao.rds.TxPipeline()
	// frontier
	pipeliner.HSet(context.TODO(), getFrontierKey(frontier),
		"edge_count", 0)
	pipeliner.Del(context.TODO(), getAliveFrontierKey(frontier))

	_, err := pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao del frontier, pipeliner exec err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) CountFrontiers() (int, error) {
	keys, err := dao.rds.Keys(context.TODO(), frontiersKeyPrefixAll).Result()
	if err != nil {
		klog.Errorf("dao count frontier, keys err: %s", err)
		return 0, err
	}
	return len(keys), nil
}

func getFrontierKey(frontier string) string {
	return frontiersKeyPrefix + frontier
}

func getFrontierID(frontierKey string) string {
	return strings.TrimPrefix(frontierKey, frontiersKeyPrefix)
}

func getAliveFrontierKey(frontier string) string {
	return frontiersAliveKeyPrefix + frontier
}

func redisArrayToFrontiers(keys []string, array []interface{}) ([]*Frontier, error) {
	fmt.Println(keys, array)
	frontiers := make([]*Frontier, len(keys))
	for i, elem := range array {
		pairs, ok := elem.([]interface{})
		if !ok {
			return nil, apis.ErrWrongTypeInRedis
		}
		frontier := &Frontier{
			FrontierID: getFrontierID(keys[i]),
		}
		// iterate inner array
		for j, pair := range pairs {
			key, ok := pair.(string)
			if !ok {
				return nil, apis.ErrWrongTypeInRedis
			}
			switch key {
			case "advertised_sb_addr":
				j = j + 1
				if j >= len(pairs) {
					return nil, apis.ErrWrongLengthInRedis
				}
				value, ok := pairs[j].(string)
				if !ok {
					return nil, apis.ErrWrongTypeInRedis
				}
				frontier.AdvertisedServiceboundAddr = value
			case "advertised_eb_addr":
				j = j + 1
				if j >= len(pairs) {
					return nil, apis.ErrWrongLengthInRedis
				}
				value, ok := pairs[j].(string)
				if !ok {
					return nil, apis.ErrWrongTypeInRedis
				}
				frontier.AdvertisedEdgeboundAddr = value
			case "edge_count":
				j = j + 1
				if j >= len(pairs) {
					return nil, apis.ErrWrongLengthInRedis
				}
				value, ok := pairs[j].(string)
				if !ok {
					return nil, apis.ErrWrongTypeInRedis
				}
				edgeCount, err := strconv.Atoi(value)
				if err != nil {
					return nil, err
				}
				frontier.EdgeCount = edgeCount
			case "service_count":
				j = j + 1
				if j >= len(pairs) {
					return nil, apis.ErrWrongLengthInRedis
				}
				value, ok := pairs[j].(string)
				if !ok {
					return nil, apis.ErrWrongTypeInRedis
				}
				serviceCount, err := strconv.Atoi(value)
				if err != nil {
					return nil, err
				}
				frontier.ServiceCount = serviceCount
			}
			frontiers[i] = frontier
		}
	}
	return frontiers, nil
}
