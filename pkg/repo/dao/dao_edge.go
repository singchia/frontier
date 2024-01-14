package dao

import (
	"github.com/singchia/frontier/pkg/repo/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type EdgeQuery struct {
	Query
	// Condition fields
	Meta   string
	Addr   string
	RPC    string
	EdgeID uint64
}

func (dao *Dao) ListEdges(query *EdgeQuery) ([]*model.Edge, error) {
	tx := dao.dbEdge.Model(&model.Edge{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildEdgeQuery(tx, query)

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.PageSize)

	// order
	if query.Order == "" {
		// desc by create_time by default
		query.Order = "edges.create_time"
		query.Desc = true
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.Order},
		Desc:   query.Desc,
	})

	// find
	edges := []*model.Edge{}
	tx = tx.Find(&edges)
	return edges, tx.Error
}

func (dao *Dao) CountEdges(query *EdgeQuery) (int64, error) {
	tx := dao.dbEdge.Model(&model.Edge{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildEdgeQuery(tx, query)

	var count int64
	tx = tx.Count(&count)
	return count, tx.Error
}

func (dao *Dao) GetEdge(edgeID uint64) (*model.Edge, error) {
	tx := dao.dbEdge.Model(&model.Edge{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = tx.Where("edge_id = ?", edgeID)

	var edge model.Edge
	tx = tx.First(&edge)
	return &edge, tx.Error
}

type EdgeDelete struct {
	EdgeID uint64
	Addr   string
}

func (dao *Dao) DeleteEdge(delete *EdgeDelete) error {
	tx := dao.dbEdge
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildEdgeDelete(tx, delete)
	return tx.Delete(&model.Edge{}).Error
}

func (dao *Dao) CreateEdge(edge *model.Edge) error {
	tx := dao.dbEdge
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	return tx.Create(edge).Error
}

func buildEdgeQuery(tx *gorm.DB, query *EdgeQuery) *gorm.DB {
	// join
	if query.RPC != "" {
		tx = tx.InnerJoins("INNER JOIN edge_rpcs ON edges.edge_id = edge_rpcs.edge_id AND service_rpcs.rpc = ?", query.RPC)
	}
	// search
	if query.Meta != "" {
		tx = tx.Where("meta LIKE ?", query.Meta+"%")
	}
	if query.Addr != "" {
		tx = tx.Where("addr LIKE ?", query.Addr+"%")
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.EdgeID != 0 {
		tx = tx.Where("edge_id = ?", query.EdgeID)
	}
	return tx
}

func buildEdgeDelete(tx *gorm.DB, delete *EdgeDelete) *gorm.DB {
	if delete.EdgeID != 0 {
		tx = tx.Where("edge_id = ?", delete.EdgeID)
	}
	if delete.Addr != "" {
		tx = tx.Where("addr = ?", delete.Addr)
	}
	return tx
}

type EdgeRPCQuery struct {
	Query
	// Condition fields
	Meta   string
	EdgeID uint64
}

func (dao *Dao) ListEdgeRPCs(query *EdgeRPCQuery) ([]string, error) {
	tx := dao.dbEdge.Model(&model.EdgeRPC{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildEdgeRPCQuery(tx, query)

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.PageSize)

	// order
	if query.Order == "" {
		// desc by create_time by default
		query.Order = "edge_rpcs.create_time"
		query.Desc = true
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.Order},
		Desc:   query.Desc,
	})

	// find
	rpcs := []string{}
	tx = tx.Distinct("rpc").Find(&rpcs)
	return rpcs, tx.Error
}

func (dao *Dao) CountEdgeRPCs(query *EdgeRPCQuery) (int64, error) {
	tx := dao.dbEdge.Model(&model.EdgeRPC{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildEdgeRPCQuery(tx, query)

	// count
	var count int64
	tx = tx.Distinct("rpc").Count(&count)
	return count, tx.Error
}

func (dao *Dao) DeleteEdgeRPCs(edgeID uint64) error {
	tx := dao.dbEdge.Where("edge_id = ?", edgeID)
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	return tx.Delete(&model.EdgeRPC{}).Error
}

func (dao *Dao) CreateEdgeRPC(rpc *model.EdgeRPC) error {
	tx := dao.dbEdge
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	return tx.Create(rpc).Error
}

func buildEdgeRPCQuery(tx *gorm.DB, query *EdgeRPCQuery) *gorm.DB {
	// join
	if query.Meta != "" {
		tx = tx.InnerJoins("INNER JOIN edges ON edges.edge_id = edge_rpcs.edge_id AND meta LIKE ?", "%"+query.Meta+"%")
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.EdgeID != 0 {
		tx = tx.Where("edge_rpcs.edge_id = ?", query.EdgeID)
	}
	return tx
}
