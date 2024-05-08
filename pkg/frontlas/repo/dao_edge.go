package repo

import (
	"context"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

// we set expire time to indicate real stats of edges
const (
	edgesKeyPrefix    = "frontlas:edges:" // example: frontlas:edges:123 "{}"
	edgesKeyPrefixAll = "frontlas:edges:*"

	edgesAliveKeyPrefix = "frontlas:alive:edges:" // example: frontlas:alive:edges:123 1 ex 30
)

//go:embed lua/edge_delete.lua
var deleteEdgeScript string

// care about the performance
func (dao *Dao) GetAllEdgeIDs() ([]uint64, error) {
	results, err := dao.rds.Keys(context.TODO(), edgesKeyPrefixAll).Result()
	if err != nil {
		klog.Errorf("dao get all edgeIDs, keys err: %s", err)
		return nil, err
	}
	edgeIDs := []uint64{}
	for _, v := range results {
		edgeID, err := getEdgeID(v)
		if err != nil {
			klog.Errorf("dao get all edgeIDs, get edgeID err: %s", err)
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

func (dao *Dao) GetEdges(edgeIDs []uint64) ([]*Edge, error) {
	keys := make([]string, len(edgeIDs))
	for i, edgeID := range edgeIDs {
		keys[i] = getEdgeKey(edgeID)
	}

	results, err := dao.rds.MGet(context.TODO(), keys...).Result()
	if err != nil {
		klog.Errorf("dao get edges, mget err: %s", err)
		return nil, err
	}
	edges := []*Edge{}
	for i, result := range results {
		if result == nil {
			edges[i] = nil
		}
		edge := &Edge{}
		err = json.Unmarshal([]byte(result.(string)), edge)
		if err != nil {
			klog.Errorf("dao get edges, json unmarshal err: %s", err)
			return nil, err
		}
	}
	return edges, nil
}

func (dao *Dao) GetEdge(edgeID uint64) (*Edge, error) {
	result, err := dao.rds.Get(context.TODO(), getEdgeKey(edgeID)).Result()
	if err != nil {
		klog.Errorf("dao get edge, get err: %s", err)
		return nil, err
	}
	edge := &Edge{}
	err = json.Unmarshal([]byte(result), edge)
	if err != nil {
		klog.Errorf("dao get edge, json unmarshal err: %s", err)
		return nil, err
	}
	return edge, nil
}

// obsoleted
func (dao *Dao) SetEdge(edgeID uint64, edge *Edge) error {
	data, err := json.Marshal(edge)
	if err != nil {
		klog.Errorf("dao set edge, json marshal err: %s", err)
		return err
	}
	_, err = dao.rds.Set(context.TODO(), getEdgeKey(edgeID), data, -1).Result()
	if err != nil {
		klog.Errorf("dao set edge, set err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) SetEdgeAndAlive(edgeID uint64, edge *Edge, expiration time.Duration) error {
	edgeData, err := json.Marshal(edge)
	if err != nil {
		klog.Errorf("dao set edge and alive, json marshal err: %s", err)
		return err
	}

	pipeliner := dao.rds.TxPipeline()
	// edge meta TODO expiration to custom
	pipeliner.Set(context.TODO(), getEdgeKey(edgeID), edgeData,
		time.Duration(dao.conf.FrontierManager.Expiration.ServiceMeta)*time.Second)
	// alive
	pipeliner.Set(context.TODO(), getAliveEdgeKey(edgeID), 1, expiration)
	// frontier edge_count
	pipeliner.HIncrBy(context.TODO(), getFrontierKey(edge.FrontierID), "edge_count", 1)

	_, err = pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao set edge and alive, pipeliner exec err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) ExpireEdge(edgeID uint64, expiration time.Duration) error {
	pipeliner := dao.rds.TxPipeline()
	// edge meta TODO expiration to custom
	pipeliner.Expire(context.TODO(), getEdgeKey(edgeID),
		time.Duration(dao.conf.FrontierManager.Expiration.ServiceMeta)*time.Second)
	// edge alive
	pipeliner.Expire(context.TODO(), getAliveEdgeKey(edgeID), expiration)

	cmds, err := pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao expire edge, pipeliner err: %s", err)
		return err
	}
	for _, cmd := range cmds {
		if cmd.Err() != nil {
			return cmd.Err()
		}
	}
	return nil
}

// we keep edge but delete alive:edge
func (dao *Dao) DeleteEdge(edgeID uint64) error {
	_, err := dao.rds.Eval(context.TODO(), deleteEdgeScript,
		[]string{getEdgeKey(edgeID), getAliveEdgeKey(edgeID), frontiersKeyPrefix}).Result()
	if err != nil {
		klog.Errorf("dao delete edge, eval err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) CountEdges() (int, error) {
	frontiers, err := dao.GetAllFrontiers()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, frontier := range frontiers {
		count += frontier.EdgeCount
	}
	return count, nil
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
