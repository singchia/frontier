package model

const (
	TnClients = "clients"
)

type Client struct {
	ClientID   uint64 `gorm:"column:client_id;primaryKey"`
	Meta       string `gorm:"column:meta;index:idx_meta"`
	Addr       string `gorm:"column:addr;index:idx_addr"`
	CreateTime int64  `gorm:"column:create_time;index:create_time"`
}

func (Client) TableName() string {
	return TnClients
}
