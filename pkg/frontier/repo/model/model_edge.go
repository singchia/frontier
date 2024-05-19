package model

const (
	TnEdges    = "edges"
	TnEdgeRPCs = "edge_rpcs"
)

type Edge struct {
	EdgeID     uint64 `gorm:"column:edge_id;primaryKey" json:"edge_id"`
	Meta       string `gorm:"column:meta;index:idx_edge_meta;type:text collate nocase" json:"meta"`
	Addr       string `gorm:"column:addr;index:idx_edge_addr;type:text collate nocase" json:"addr"`
	CreateTime int64  `gorm:"column:create_time;index:idx_edge_create_time" json:"create_time"`
}

func (Edge) TableName() string {
	return TnEdges
}

type EdgeRPC struct {
	RPC        string `gorm:"column:rpc;index:idx_edgerpc_rpc" json:"rpc"`
	EdgeID     uint64 `gorm:"column:edge_id;index:idx_edgerpc_edge_id" json:"edge_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_edgerpc_create_time" json:"create_time"`
}

func (EdgeRPC) TableName() string {
	return TnEdgeRPCs
}
