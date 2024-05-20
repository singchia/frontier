package membuntdb

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/singchia/frontier/pkg/frontier/misc"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/tidwall/buntdb"
)

func (dao *dao) ListEdges(query *query.EdgeQuery) ([]*model.Edge, error) {
	var (
		offset, size int
		idx, pivot   string
		desc         bool
		field        int
	)
	if query.RPC != "" {
		field |= 1
	}
	if query.Addr != "" {
		field |= 2
		pivot = fmt.Sprintf(`{"addr": %s}`, query.Addr)
	}
	if query.Meta != "" {
		field |= 4
		pivot = fmt.Sprintf(`{"meta": %s}`, query.Meta)
	}
	if field&(field-1) != 0 {
		// multiple fields set
		// TODO
	}

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset = query.PageSize * (query.Page - 1)

	// order and index
	switch query.Order {
	case "addr":
		idx = IdxEdge_Addr
		desc = query.Desc
	case "meta":
		idx = IdxEdge_Meta
		desc = query.Desc
	default:
		// desc by create_time by default
		query.Order = IdxEdgeRPC_CreateTime
		desc = query.Desc
	}

	skip := 0
	edges := []*model.Edge{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		var (
			err     error
			finderr error
		)
		find := tx.DescendGreaterThan
		if !desc {
			find = tx.AscendGreaterOrEqual
		}
		finderr = find(idx, pivot, func(key, value string) bool {
			if skip < offset {
				skip += 1
				return true
			} else if skip > offset+size {
				return false
			}
			edge := &model.Edge{}
			err = json.Unmarshal([]byte(value), edge)
			if err != nil {
				return false
			}
			edges = append(edges, edge)
			skip += 1
			return true
		})
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
	if query.Meta != "" {
		return nil, ErrUnsupportedForBuntDB
	}

	rpcs := map[string]struct{}{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		var (
			err     error
			finderr error
		)
		finderr = tx.AscendGreaterOrEqual("", getEdgeRPCPrefixKey(query.EdgeID), func(key, value string) bool {
			edgeRPC := &model.EdgeRPC{}
			err = json.Unmarshal([]byte(value), edgeRPC)
			if err != nil {
				return false
			}
			rpcs[edgeRPC.RPC] = struct{}{}
			return true
		})
		if err != nil {
			return err
		}
		if finderr != nil {
			return finderr
		}
		return nil
	})
	return misc.GetKeys(rpcs), err
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

func (dao *dao) listEdgeRPCsByRPC(query *query.EdgeRPCQuery) ([]*model.EdgeRPC, error) {

}

func getEdgeRPCKey(edgeID uint64, rpc string) string {
	return "edge_rpcs:" + strconv.FormatUint(edgeID, 10) + "-" + rpc
}

func getEdgeRPCPrefixKey(edgeID uint64) string {
	if edgeID != 0 {
		return "edge_rpcs:" + strconv.FormatUint(edgeID, 10)
	}
	return "edge_rpcs:"
}
