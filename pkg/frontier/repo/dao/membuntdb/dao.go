package membuntdb

import (
	"errors"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/tidwall/buntdb"
	"k8s.io/klog/v2"
)

const (
	IdxEdge_Meta               = "idx_edge_meta"
	IdxEdge_Addr               = "idx_edge_addr"
	IdxEdge_CreateTime         = "idx_create_time"
	IdxEdgeRPC_RPC             = "idx_edgerpc_rpc"
	IdxEdgeRPC_EdgeID          = "idx_edgerpc_edge_id"
	IdxEdgeRPC_CreateTime      = "idx_edgerpc_create_time"
	IdxService_Service         = "idx_service_service"
	IdxService_Addr            = "idx_service_addr"
	IdxService_CreateTime      = "index_service_create_time"
	IdxServiceRPC_RPC          = "idx_servicerpc_rpc"
	IdxServiceRPC_ServiceID    = "idx_servicerpc_service_id"
	IdxServiceRPC_CreateTime   = "idx_servicerpc_create_time"
	IdxServiceTopic_Topic      = "idx_servicetopic_topic"
	IdxServiceTopic_ServiceID  = "idx_servicetopic_service_id"
	IdxServiceTopic_CreateTime = "idx_servicetopic_create_time"
)

var (
	ErrUnimplemented        = errors.New("unimplemented")
	ErrUnsupportedForBuntDB = errors.New("unsupported for buntdb")
)

type dao struct {
	db     *buntdb.DB
	config *config.Configuration
}

func NewDao(config *config.Configuration) (*dao, error) {
	db, err := buntdb.Open(":memory:")
	if err != nil {
		klog.Errorf("dao open buntdb err: %s", err)
		return nil, err
	}
	db.SetConfig(buntdb.Config{})
	// edge's indexes
	err = db.CreateIndex(IdxEdge_Meta, "edges*", buntdb.IndexJSON("meta"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxEdge_Addr, "edges*", buntdb.IndexJSON("addr"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxEdge_CreateTime, "edges*", buntdb.IndexJSON("create_time"))
	if err != nil {
		return nil, err
	}
	// edgeRPC's indexes
	err = db.CreateIndex(IdxEdgeRPC_RPC, "edge_rpcs*", buntdb.IndexJSON("rpc"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxEdgeRPC_EdgeID, "edge_rpcs*", buntdb.IndexJSON("edge_id"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxEdgeRPC_CreateTime, "edge_rpcs*", buntdb.IndexJSON("create_time"))
	if err != nil {
		return nil, err
	}
	// service's indexes
	err = db.CreateIndex(IdxService_Service, "services*", buntdb.IndexJSON("service"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxService_Addr, "services*", buntdb.IndexJSON("addr"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxService_CreateTime, "services*", buntdb.IndexJSON("create_time"))
	if err != nil {
		return nil, err
	}
	// serviceRPC's indexes
	err = db.CreateIndex(IdxServiceRPC_RPC, "service_rpcs*", buntdb.IndexJSON("rpc"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxServiceRPC_ServiceID, "service_rpcs*", buntdb.IndexJSON("service_id"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxServiceRPC_CreateTime, "service_rpcs*", buntdb.IndexJSON("create_time"))
	if err != nil {
		return nil, err
	}
	// serviceTopics's indexes
	err = db.CreateIndex(IdxServiceTopic_Topic, "service_topics*", buntdb.IndexJSON("topic"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxServiceTopic_ServiceID, "service_topics*", buntdb.IndexJSON("service_id"))
	if err != nil {
		return nil, err
	}
	err = db.CreateIndex(IdxServiceTopic_CreateTime, "service_topics*", buntdb.IndexJSON("create_time"))
	if err != nil {
		return nil, err
	}
	return &dao{
		db:     db,
		config: config,
	}, nil
}

func (dao *dao) Close() error {
	return dao.db.Close()
}
