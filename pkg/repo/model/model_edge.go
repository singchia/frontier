package model

const (
	TnEdges    = "clients"
	TnEdgeRPCs = "client_rpcs"
)

type Edge struct {
	EdgeID     uint64 `gorm:"column:client_id;primaryKey"`
	Meta       string `gorm:"column:meta;index:idx_meta;type:text collate nocase"`
	Addr       string `gorm:"column:addr;index:idx_addr;type:text collate nocase"`
	CreateTime int64  `gorm:"column:create_time;index:idx_create_time"`
}

func (Edge) TableName() string {
	return TnEdges
}

type EdgeRPC struct {
	RPC        string `gorm:"column:rpc;index:idx_rpc"`
	EdgeID     uint64 `gorm:"column:client_id;index:idx_client_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_create_time"`
}

func (EdgeRPC) TableName() string {
	return TnEdgeRPCs
}
