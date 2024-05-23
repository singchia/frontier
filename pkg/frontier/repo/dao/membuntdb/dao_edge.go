package membuntdb

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

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
		// TODO we must search index rpc first
		return nil, ErrUnsupportedForBuntDB
	}
	if query.Addr != "" {
		field |= 2
	}
	if query.Meta != "" {
		field |= 4
	}
	if field&(field-1) != 0 {
		// multiple fields set
		return nil, ErrUnsupportedMultipleFieldsForBuntDB
	}

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset = query.PageSize * (query.Page - 1)
	size = query.PageSize

	// order and index
	switch query.Order {
	case "addr":
		if query.Meta != "" {
			return nil, ErrUnsupportedForBuntDB
		}
		idx = IdxEdge_Addr
		desc = query.Desc
		pivot = fmt.Sprintf(`{"addr": "%s"}`, query.Addr)
	case "meta":
		if query.Addr != "" {
			return nil, ErrUnsupportedForBuntDB
		}
		idx = IdxEdge_Meta
		desc = query.Desc
		pivot = fmt.Sprintf(`{"meta": "%s"}`, query.Meta)
	default:
		// desc by create_time by default
		idx = IdxEdge_CreateTime
		desc = true
	}

	edges := []*model.Edge{}
	switch idx {
	case IdxEdge_Addr, IdxEdge_Meta:
		// index on addr or meta
		err := dao.db.View(func(tx *buntdb.Tx) error {
			find := tx.DescendGreaterThan // TODO range [gte, lt)
			if !desc {
				find = tx.AscendGreaterOrEqual
			}
			skip := 0
			finderr := find(idx, pivot, func(key, value string) bool {
				edge, keepon, err := edgeMatch(query.Addr, query.Meta, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
				if err != nil {
					// TODO
					return true
				}
				if edge != nil {
					edges = append(edges, edge)
				}
				return keepon
			})
			return finderr
		})
		return edges, err

	default:
		// index on create_time
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.StartTime != 0 && query.EndTime != 0 {
				find := tx.DescendRange
				if desc {
					query.StartTime -= 1
				} else if !desc {
					find = tx.AscendRange
				}
				less := fmt.Sprintf(`{"create_time": %d}`, query.EndTime)
				greater := fmt.Sprintf(`{"create_time": %d}`, query.StartTime)
				skip := 0
				finderr := find(idx, less, greater, func(key, value string) bool {
					edge, keepon, err := edgeMatch(query.Addr, query.Meta, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if edge != nil {
						edges = append(edges, edge)
					}
					return keepon
				})
				return finderr

			} else {
				find := tx.Descend
				if !desc {
					find = tx.Ascend
				}
				skip := 0
				finderr := find(idx, func(key, value string) bool {
					edge, keepon, err := edgeMatch(query.Addr, query.Meta, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if edge != nil {
						edges = append(edges, edge)
					}
					return keepon
				})
				return finderr
			}
		})
		return edges, err
	}
}

func edgeMatch(addr string, meta string, startTime int64, endTime int64, offset int, size int, skip *int, desc bool, edgeStr string) (*model.Edge, bool, error) {
	edge := &model.Edge{}
	err := json.Unmarshal([]byte(edgeStr), edge)
	if err != nil {
		// TODO
		return nil, true, err
	}
	// pattern unmatch
	if (addr != "" && !strings.HasPrefix(edge.Addr, addr)) ||
		(meta != "" && !strings.HasPrefix(edge.Meta, meta)) {
		if desc {
			// example: test1 test2 test3 test4 foo5, and search on prefix "test" descent, we won't match, but still need to iterate
			return nil, true, nil
		}
		return nil, false, nil
	}
	// time range unmatch
	if startTime != 0 && endTime != 0 && (edge.CreateTime < startTime || edge.CreateTime >= endTime) {
		// continue
		return nil, true, nil
	}
	// offset and size
	defer func() { *skip = *skip + 1 }()
	if *skip < offset {
		return nil, true, nil
	} else if *skip >= offset+size {
		// break out
		return nil, false, nil
	} else {
		return edge, true, nil
	}
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
	var (
		offset, size int
		idx, pivot   string
		desc         bool
	)

	if query.Meta != "" {
		// TODO we must search index edge first
		return nil, ErrUnsupportedForBuntDB
	}

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset = query.PageSize * (query.Page - 1)
	size = query.PageSize

	// order and index
	switch query.Order {
	case "edge_id":
		idx = IdxEdgeRPC_EdgeID
		desc = query.Desc
		pivot = fmt.Sprintf(`{"edge_id": %d}`, query.EdgeID)
	default:
		// desc by create_time by default
		idx = IdxEdgeRPC_CreateTime
		desc = true
	}

	rpcs := map[string]struct{}{}
	switch idx {
	case IdxEdgeRPC_EdgeID:
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.EdgeID == 0 {
				find := tx.DescendGreaterThan
				if !desc {
					find = tx.AscendGreaterOrEqual
				}
				skip := 0
				finderr := find(idx, pivot, func(key, value string) bool {
					edgeRPC, keepon, err := edgeRPCMatch(0, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if edgeRPC != nil {
						rpcs[edgeRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr
			} else {
				find := tx.DescendEqual
				if !desc {
					find = tx.AscendEqual
				}
				skip := 0
				finderr := find(idx, pivot, func(key, value string) bool {
					edgeRPC, keepon, err := edgeRPCMatch(query.EdgeID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						return true
					}
					if edgeRPC != nil {
						rpcs[edgeRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr
			}
		})
		return misc.GetKeys(rpcs), err

	default:
		// index on create_time
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.StartTime != 0 && query.EndTime != 0 {
				find := tx.DescendRange
				if desc {
					query.StartTime -= 1
				} else if !desc {
					find = tx.AscendRange
				}
				less := fmt.Sprintf(`{"create_time": %d}`, query.EndTime)
				greater := fmt.Sprintf(`{"create_time": %d}`, query.StartTime)
				skip := 0
				finderr := find(idx, less, greater, func(key, value string) bool {
					edgeRPC, keepon, err := edgeRPCMatch(query.EdgeID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if edgeRPC != nil {
						rpcs[edgeRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr

			} else {
				find := tx.Descend
				if !desc {
					find = tx.Ascend
				}
				skip := 0
				finderr := find(idx, func(key, value string) bool {
					edgeRPC, keepon, err := edgeRPCMatch(query.EdgeID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if edgeRPC != nil {
						rpcs[edgeRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr
			}
		})
		return misc.GetKeys(rpcs), err
	}
}

func edgeRPCMatch(edgeID uint64, startTime int64, endTime int64, offset int, size int, skip *int, desc bool, edgeRPCStr string) (*model.EdgeRPC, bool, error) {
	edgeRPC := &model.EdgeRPC{}
	err := json.Unmarshal([]byte(edgeRPCStr), edgeRPC)
	if err != nil {
		// TODO
		return nil, true, err
	}
	if edgeID != 0 && edgeID != edgeRPC.EdgeID {
		// if we use DescendEqual or AscendEqual, it won't hit here
		if desc {
			return nil, true, nil
		}
		return nil, false, nil
	}
	// time range unmatch
	if startTime != 0 && endTime != 0 && (edgeRPC.CreateTime < startTime || edgeRPC.CreateTime >= endTime) {
		// continue
		return nil, true, nil
	}
	// offset and size
	defer func() { *skip = *skip + 1 }()
	if *skip < offset {
		return nil, true, nil
	} else if *skip >= offset+size {
		// break out
		return nil, false, nil
	} else {
		return edgeRPC, true, nil
	}
}

func (dao *dao) CountEdgeRPCs(query *query.EdgeRPCQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) DeleteEdgeRPCs(edgeID uint64) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		var delkeys []string
		pivot := fmt.Sprintf(`{"edge_id": %d}`, edgeID)
		tx.AscendEqual(IdxEdgeRPC_EdgeID, pivot, func(key, value string) bool {
			delkeys = append(delkeys, key)
			return true
		})
		for _, key := range delkeys {
			if _, err := tx.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
	return err
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

func getEdgeRPCPrefixKey(edgeID uint64) string {
	if edgeID != 0 {
		return "edge_rpcs:" + strconv.FormatUint(edgeID, 10)
	}
	return "edge_rpcs:"
}
