package dao

import (
	"github.com/singchia/frontier/pkg/repo/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClientQuery struct {
	// Pagination
	Page, PageSize int
	// Time range
	StartTime, EndTime int64
	// Search fields
	Meta     string
	Addr     string
	RPC      string
	ClientID uint64
	// Order
	Order string
	Desc  bool
}

type Client struct {
	ClientID   uint64
	Meta       string
	Addr       string
	CreateTime int64
}

func (dao *Dao) ListClients(query *ClientQuery) ([]*Client, error) {
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
	clients := []*Client{}
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

func (dao *Dao) GetClient(clientID uint64) (*Client, error) {
	tx := dao.db.Model(&model.Client{})
	tx.Where("client_id = ?", clientID)
	var client Client
	result := tx.First(&client)
	return &client, result.Error
}

func (dao *Dao) DeleteClient(clientID uint64) error {
	return dao.db.Where("client_id = ?", clientID).Delete(&model.Client{}).Error
}

func (dao *Dao) CreateClient(client *Client) error {
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
