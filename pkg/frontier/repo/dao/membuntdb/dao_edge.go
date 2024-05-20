package membuntdb

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/tidwall/buntdb"
)

func (dao *dao) ListEdges(query *query.EdgeQuery) ([]*model.Edge, error) {
	edges := []*model.Edge{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		var (
			err     error
			finderr error
		)
		// query on addr
		if query.Addr != "" {
			pivot := fmt.Sprintf(`{"addr":%s}`, query.Addr)
			finderr = tx.AscendGreaterOrEqual(IdxEdge_Addr, pivot, func(key, value string) bool {
				edge := &model.Edge{}
				err = json.Unmarshal([]byte(value), edge)
				if err != nil {
					return false
				}
				if !strings.HasPrefix(edge.Addr, query.Addr) {
					return false
				}
				edges = append(edges, edge)
				return true
			})
		}
		if err != nil {
			return err
		}
		if finderr != nil {
			return finderr
		}
		return nil
	})
	return edges, err
}

func (dao *dao) CountEdges(query *query.EdgeQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) GetEdge(edgeID uint64) (*model.Edge, error) {
	edge := &model.Edge{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(getEdgeKey(edgeID))
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(value), edge)
		return err
	})
	return edge, err
}

func (dao *dao) DeleteEdge(delete *query.EdgeDelete) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		if delete.EdgeID != 0 {
			_, err := tx.Delete(getEdgeKey(delete.EdgeID))
			return err
		}
		return nil
	})
	return err
}

func (dao *dao) CreateEdge(edge *model.Edge) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(edge)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(getEdgeKey(edge.EdgeID), string(data), nil)
		return err
	})
	return err
}

func getEdgeKey(edgeID uint64) string {
	return "edges:" + strconv.FormatUint(edgeID, 10)
}

func (dao *dao) ListEdgeRPCs(query *query.EdgeRPCQuery) ([]string, error) {
	return nil, ErrUnimplemented
}

func (dao *dao) CountEdgeRPCs(query *query.EdgeRPCQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) DeleteEdgeRPCs(edgeID uint64) error {
	return nil
}

func (dao *dao) CreateEdgeRPC(rpc *model.EdgeRPC) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(rpc)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(getEdgeRPCKey(rpc.EdgeID, rpc.RPC), string(data), nil)
		return err
	})
	return err
}

func getEdgeRPCKey(edgeID uint64, rpc string) string {
	return "edge_rpcs:" + strconv.FormatUint(edgeID, 10) + "-" + rpc
}
