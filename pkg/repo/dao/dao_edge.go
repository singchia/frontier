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
		query.Order = "create_time"
		query.Desc = true
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.Order},
		Desc:   query.Desc,
	})

	// find
	clients := []*model.Edge{}
	result := tx.Find(&clients)
	return clients, result.Error
}

func (dao *Dao) CountEdges(query *EdgeQuery) (int64, error) {
	tx := dao.dbEdge.Model(&model.Edge{})
	tx = buildEdgeQuery(tx, query)

	// count
	var count int64
	result := tx.Count(&count)
	return count, result.Error
}

func (dao *Dao) GetEdge(clientID uint64) (*model.Edge, error) {
	tx := dao.dbEdge.Model(&model.Edge{})
	tx.Where("client_id = ?", clientID)
	var client model.Edge
	result := tx.First(&client)
	return &client, result.Error
}

func (dao *Dao) DeleteEdge(clientID uint64) error {
	return dao.dbEdge.Where("client_id = ?", clientID).Delete(&model.Edge{}).Error
}

func (dao *Dao) CreateEdge(client *model.Edge) error {
	return dao.dbEdge.Create(client).Error
}

func buildEdgeQuery(tx *gorm.DB, query *EdgeQuery) *gorm.DB {
	// join
	if query.RPC != "" {
		tx = tx.InnerJoins("INNER JOIN client_rpcs ON clients.client_id = client_rpcs.client_id AND rpc = ?", query.RPC)
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
		tx = tx.Where("client_id = ?", query.EdgeID)
	}
	return tx
}

type EdgeRPCQuery struct {
	Query
	// Condition fields
	Meta   string
	EdgeID uint64
}

// list RPCs doesn't handle order
func (dao *Dao) ListEdgeRPCs(query *EdgeRPCQuery) ([]string, error) {
	tx := dao.dbEdge.Model(&model.EdgeRPC{})
	tx = buildEdgeRPCQuery(tx, query)
	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.PageSize)

	rpcs := []string{}
	result := tx.Distinct("rpc").Find(&rpcs)
	return rpcs, result.Error
}

func (dao *Dao) CountEdgeRPCs(query *EdgeRPCQuery) (int64, error) {
	tx := dao.dbEdge.Model(&model.EdgeRPC{})
	tx = buildEdgeRPCQuery(tx, query)

	// count
	var count int64
	result := tx.Distinct("rpc").Count(&count)
	return count, result.Error
}

func (dao *Dao) DeleteEdgeRPCs(clientID uint64) error {
	return dao.dbEdge.Where("client_id = ?", clientID).Delete(&model.EdgeRPC{}).Error
}

func (dao *Dao) CreateEdgeRPC(rpc *model.EdgeRPC) error {
	return dao.dbEdge.Create(rpc).Error
}

func buildEdgeRPCQuery(tx *gorm.DB, query *EdgeRPCQuery) *gorm.DB {
	// join
	if query.Meta != "" {
		tx = tx.InnerJoins("INNER JOIN clients ON clients.client_id = client_rpcs.client_id AND meta like ?", "%"+query.Meta+"%")
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.EdgeID != 0 {
		tx = tx.Where("client_rpcs.client_id = ?", query.EdgeID)
	}
	return tx
}
