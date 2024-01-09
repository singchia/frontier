package dao

import (
	"github.com/singchia/frontier/pkg/repo/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClientQuery struct {
	Query
	// Condition fields
	Meta     string
	Addr     string
	RPC      string
	ClientID uint64
}

func (dao *Dao) ListClients(query *ClientQuery) ([]*model.Client, error) {
	tx := dao.db.Model(&model.Client{})
	tx = buildClientQuery(tx, query)

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.Page)

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
	clients := []*model.Client{}
	result := tx.Find(&clients)
	return clients, result.Error
}

func (dao *Dao) CountClients(query *ClientQuery) (int64, error) {
	tx := dao.db.Model(&model.Client{})
	tx = buildClientQuery(tx, query)

	// count
	var count int64
	result := tx.Count(&count)
	return count, result.Error
}

func (dao *Dao) GetClient(clientID uint64) (*model.Client, error) {
	tx := dao.db.Model(&model.Client{})
	tx.Where("client_id = ?", clientID)
	var client model.Client
	result := tx.First(&client)
	return &client, result.Error
}

func (dao *Dao) DeleteClient(clientID uint64) error {
	return dao.db.Where("client_id = ?", clientID).Delete(&model.Client{}).Error
}

func (dao *Dao) CreateClient(client *model.Client) error {
	return dao.db.Create(client).Error
}

func buildClientQuery(tx *gorm.DB, query *ClientQuery) *gorm.DB {
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
	if query.ClientID != 0 {
		tx = tx.Where("client_id = ?", query.ClientID)
	}
	return tx
}

type ClientRPCQuery struct {
	Query
	// Condition fields
	Meta     string
	ClientID uint64
}

// list RPCs doesn't handle order
func (dao *Dao) ListClientRPCs(query *ClientRPCQuery) ([]string, error) {
	tx := dao.db.Model(&model.ClientRPC{})
	tx = buildClientRPCQuery(tx, query)
	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.Page)

	rpcs := []string{}
	result := tx.Distinct("rpc").Find(&rpcs)
	return rpcs, result.Error
}

func (dao *Dao) CountClientRPCs(query *ClientRPCQuery) (int64, error) {
	tx := dao.db.Model(&model.ClientRPC{})
	tx = buildClientRPCQuery(tx, query)

	// count
	var count int64
	result := tx.Distinct("rpc").Count(&count)
	return count, result.Error
}

func (dao *Dao) DeleteClientRPCs(clientID uint64) error {
	return dao.db.Where("client_id = ?", clientID).Delete(&model.ClientRPC{}).Error
}

func (dao *Dao) CreateClientRPC(rpc *model.ClientRPC) error {
	return dao.db.Create(rpc).Error
}

func buildClientRPCQuery(tx *gorm.DB, query *ClientRPCQuery) *gorm.DB {
	// join
	if query.Meta != "" {
		tx = tx.InnerJoins("INNER JOIN clients ON clients.client_id = client_rpcs.client_id AND meta like ?", "%"+query.Meta+"%")
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.ClientID != 0 {
		tx = tx.Where("client_rpcs.client_id = ?", query.ClientID)
	}
	return tx
}
