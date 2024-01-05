package dao

import (
	"github.com/singchia/frontier/internal/repo/model"
	"gorm.io/gorm"
)

type ClientQuery struct {
	Page, PageSize     int
	StartTime, EndTime int64
	Meta               string
	Addr               string
	RPC                string
	ClientID           uint64
	Ordering           string
}

type Client struct {
	ClientID   uint64
	Meta       string
	Addr       string
	CreateTime int64
}

func (dao *Dao) ListClients(query *ClientQuery) ([]*Client, error) {
	tx := dao.db.Model(model.Client{})
	tx = buildClientQuery(tx, query)
	return nil, nil
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
