package model

const (
	TnServices      = "services"
	TnServiceRPCs   = "service_rpcs"
	TnServiceTopics = "service_topics"
)

type Service struct {
	ServiceID  uint64 `gorm:"column:service_id;primaryKey" json:"service_id"`
	Service    string `gorm:"column:service;index:idx_service_service;type:text collate nocase" json:"service"`
	Addr       string `gorm:"column:addr;index:idx_service_addr;type:text collate nocase" json:"addr"`
	CreateTime int64  `gorm:"column:create_time;index_service_create_time" json:"create_time"`
}

func (Service) TableName() string {
	return TnServices
}

type ServiceRPC struct {
	RPC        string `gorm:"column:rpc;index:idx_servicerpc_rpc;type:text collate nocase" json:"rpc"`
	ServiceID  uint64 `gorm:"service_id;index:idx_servicerpc_service_id" json:"service_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_servicerpc_create_time" json:"create_time"`
}

func (ServiceRPC) TableName() string {
	return TnServiceRPCs
}

type ServiceTopic struct {
	Topic      string `gorm:"column:topic;index:idx_servicetopic_topic;type:text collate nocase" json:"topic"`
	ServiceID  uint64 `gorm:"service_id;index:idx_servicetopic_service_id" json:"service_id"`
	CreateTime int64  `gorm:"column:create_time;index:idx_servicetopic_create_time" json:"create_time"`
}

func (ServiceTopic) TableName() string {
	return TnServiceTopics
}
