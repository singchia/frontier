package repo

import (
	"context"
	"strconv"

	"k8s.io/klog/v2"
)

const (
	edgesKey = "frontier:edges"
)

func (dao *Dao) GetAllEdgeIDs() ([]uint64, error) {
	results, err := dao.rds.HGetAll(context.TODO(), edgesKey).Result()
	if err != nil {
		return nil, err
	}
	edgeIDs := []uint64{}
	for k, _ := range results {
		edgeID, err := strconv.ParseUint(k, 10, 64)
		if err != nil {
			klog.Errorf("dao get all edgeIDs err: %s", err)
			return nil, err
		}
		edgeIDs = append(edgeIDs, edgeID)
	}
	return edgeIDs, nil
}
