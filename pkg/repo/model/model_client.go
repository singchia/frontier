package model

const (
	TnClients    = "clients"
	TnClientRPCs = "client_rpcs"
)

type Client struct {
	ClientID   uint64 `gorm:"column:client_id;primaryKey"`
	Meta       string `gorm:"column:meta;index:idx_meta;type:text collate nocase"`
	Addr       string `gorm:"column:addr;index:idx_addr;type:text collate nocase"`
	CreateTime int64  `gorm:"column:create_time;index:idx_create_time"`
}

func (Client) TableName() string {
	return TnClients
}

type ClientRPC struct {
	RPC        string `gorm:"column:rpc;index:idx_rpc"`
	ClientID   uint64 `gorm:"column:client_id;index:idx_client_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_create_time"`
}

func (ClientRPC) TableName() string {
	return TnClientRPCs
}
