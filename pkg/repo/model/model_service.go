package model

const (
	TnServices      = "services"
	TnServiceRPCs   = "service_rpcs"
	TnServiceTopics = "service_topics"
)

type Service struct {
	ServiceID  uint64 `gorm:"column:service_id;primaryKey"`
	Service    string `gorm:"column:service;index:idx_service;type:text collate nocase"`
	Addr       string `gorm:"column:addr;index:idx_addr;type:text collate nocase"`
	CreateTime int64  `gorm:"column:create_time;index_create_time"`
}

func (Service) TableName() string {
	return TnServices
}

type ServiceRPC struct {
	RPC        string `gorm:"column:rpc;index:idx_rpc;type:text collate nocase"`
	ServiceID  uint64 `gorm:"service_id;index:idx_service_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_create_time"`
}

func (ServiceRPC) TableName() string {
	return TnServiceRPCs
}

type ServiceTopic struct {
	Topic      string `gorm:"column:topic;index:idx_topic;type:text collate nocase"`
	ServiceID  uint64 `gorm:"service_id;index:idx_service_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_create_time"`
}

func (ServiceTopic) TableName() string {
	return TnServiceTopics
}
