package repo

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/singchia/frontier/pkg/frontlas/apis"
	"k8s.io/klog/v2"
)

// we set expire time to indicate real stats of edges
const (
	edgesKeyPrefix    = "frontlas:edges:" // example: frontlas:alive:edges:123 "{}"
	edgesKeyPrefixAll = "frontlas:edges:*"

	edgesAliveKeyPrefix = "frontlas:alive:edges:" // example: frontlas:alive:edges:123 1 ex 20
)

// care about the performance
func (dao *Dao) GetAllEdgeIDs() ([]uint64, error) {
	results, err := dao.rds.Keys(context.TODO(), edgesKeyPrefixAll).Result()
	if err != nil {
		return nil, err
	}
	edgeIDs := []uint64{}
	for _, v := range results {
		edgeID, err := getEdgeID(v)
		if err != nil {
			klog.Errorf("dao get all edgeIDs, strconv parse err: %s, ", err)
			return nil, err
		}
		edgeIDs = append(edgeIDs, edgeID)
	}
	return edgeIDs, nil
}

type EdgeQuery struct {
	Cursor uint64
	Count  int64
}

func (dao *Dao) GetEdgesByCursor(query *EdgeQuery) ([]*Edge, uint64, error) {
	edges := []*Edge{}
	keys, cursor, err := dao.rds.Scan(context.TODO(), query.Cursor, edgesKeyPrefixAll, query.Count).Result()
	if err != nil {
		klog.Errorf("dao get edges, scan err: %s", err)
		return nil, 0, err
	}
	if keys == nil || len(keys) == 0 {
		return edges, cursor, nil
	}
	results, err := dao.rds.MGet(context.TODO(), keys...).Result()
	if err != nil {
		klog.Errorf("dao get edges, mget err: %s, keys: %v", err, keys)
		return nil, 0, err
	}
	for _, elem := range results {
		edge := &Edge{}
		err = json.Unmarshal([]byte(elem.(string)), edge)
		if err != nil {
			klog.Errorf("dao get edges, json unmarshal err: %s", err)
			return nil, 0, err
		}
		edges = append(edges, edge)
	}
	return edges, cursor, nil
}

func (dao *Dao) GetEdge(edgeID uint64) (*Edge, error) {
	result, err := dao.rds.Get(context.TODO(), getEdgeKey(edgeID)).Result()
	if err != nil {
		klog.Errorf("dao get edge, get err: %s", err)
	}
	edge := &Edge{}
	err = json.Unmarshal([]byte(result), edge)
	if err != nil {
		klog.Errorf("dao get edges, json unmarshal err: %s", err)
		return nil, err
	}
	return edge, nil
}

func (dao *Dao) SetEdge(edgeID uint64, edge *Edge) error {
	edgeKey := getEdgeKey(edgeID)
	data, err := json.Marshal(edge)
	if err != nil {
		klog.Errorf("dao set edge, json marshal err: %s", err)
		return err
	}
	_, err = dao.rds.Set(context.TODO(), edgeKey, data, -1).Result()
	if err != nil {
		klog.Errorf("dao set edge, set err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) SetEdgeAndAlive(edgeID uint64, edge *Edge, expiration time.Duration) error {
	// edge
	edgeKey := getEdgeKey(edgeID)
	edgeData, err := json.Marshal(edge)
	if err != nil {
		klog.Errorf("dao set edge and alive, json marshal err: %s", err)
		return err
	}
	// alive
	edgeAliveKey := getAliveEdgeKey(edgeID)

	pipeliner := dao.rds.TxPipeline()
	pipeliner.Set(context.TODO(), edgeKey, edgeData, -1)
	pipeliner.Set(context.TODO(), edgeAliveKey, 1, expiration)
	_, err = pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao set edge and alive, pipeliner exec err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) ExpireEdge(edgeID uint64, expiration time.Duration) error {
	edgeAliveKey := getAliveEdgeKey(edgeID)
	ok, err := dao.rds.Expire(context.TODO(), edgeAliveKey, expiration).Result()
	if err != nil {
		return err
	}
	if !ok {
		return apis.ErrExpireFailed
	}
	return nil
}

func (dao *Dao) DeleteEdge(edgeID uint64) error {
	edgeKey := getEdgeKey(edgeID)
	edgeAliveKey := getAliveEdgeKey(edgeID)

	script := `
	local edge_key = KEYS[1]
	local edge_alive_key = KEYS[2]
	
	# get edge and it's frontier_id
	local edge = redis.call("GET", edge_key)
	if edge then
		local value = json.decode(edge)
		local frontier_id = value['frontier_id']
		if frontier_id then
			# decrement the edge_count in frontier
			local frontier_key = "frontlas:frontiers:" + frontier_id
			redis.call("HINCRBY", frontier_key, "edge_count", -1)
		end
	end

	# remove edge alive
	return redis.call("DEL", edge_alive_key)
	`

	_, err := dao.rds.Eval(context.TODO(), script, []string{edgeKey, edgeAliveKey}).Result()
	if err != nil {
		klog.Errorf("dao delete edge, eval err: %s", err)
		return err
	}
	return nil
}

func getEdgeKey(edgeID uint64) string {
	return edgesKeyPrefix + strconv.FormatUint(edgeID, 10)
}

func getEdgeID(edgeKey string) (uint64, error) {
	key := strings.TrimPrefix(edgeKey, edgesKeyPrefix)
	return strconv.ParseUint(key, 10, 64)
}

func getAliveEdgeKey(edgeID uint64) string {
	return edgesAliveKeyPrefix + strconv.FormatUint(edgeID, 10)
}
