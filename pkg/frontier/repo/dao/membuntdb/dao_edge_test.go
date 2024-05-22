package membuntdb

import (
	"testing"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
)

func TestListEdges(t *testing.T) {
	config := &config.Configuration{}
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()
	edges := []*model.Edge{
		{
			EdgeID:     1,
			Meta:       "test1",
			Addr:       "192.168.1.101",
			CreateTime: 11,
		}, {
			EdgeID:     2,
			Meta:       "test2",
			Addr:       "192.168.1.102",
			CreateTime: 12,
		}, {
			EdgeID:     3,
			Meta:       "test3",
			Addr:       "192.168.1.103",
			CreateTime: 13,
		}, {
			EdgeID:     4,
			Meta:       "test4",
			Addr:       "192.168.1.104",
			CreateTime: 14,
		}, {
			EdgeID:     5,
			Meta:       "foo5",
			Addr:       "172.16.1.105",
			CreateTime: 15,
		},
	}
	for _, edge := range edges {
		err = dao.CreateEdge(edge)
		if err != nil {
			t.Error(err)
		}
	}

	// query on prefix addr
	retEdges, err := dao.ListEdges(&query.EdgeQuery{
		Addr: "192.168",
	})
	if err != nil {
		t.Error(err)
	}
	if len(retEdges) != 4 {
		t.Error("unmatched length of edges")
	}

	// query on prefix addr and create time
	retEdges, err = dao.ListEdges(&query.EdgeQuery{
		Addr: "192.168",
		Query: query.Query{
			StartTime: 12,
			EndTime:   13,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(retEdges) != 1 {
		t.Error("unmatched length of edges", len(retEdges))
	}

	// query on addr, create time and order
	retEdges, err = dao.ListEdges(&query.EdgeQuery{
		Addr: "192.168",
		Query: query.Query{
			StartTime: 11,
			EndTime:   13,
			Order:     "addr",
		},
	})
	if err != nil {
		t.Error(err)
	}
	t.Log(len(retEdges))
}

func TestListEdgeRPCs(t *testing.T) {
	config := &config.Configuration{}
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()
	edgeRPCs := []*model.EdgeRPC{
		{
			RPC:        "rpc1",
			EdgeID:     1,
			CreateTime: 11,
		},
		{
			RPC:        "rpc2",
			EdgeID:     2,
			CreateTime: 12,
		},
		{
			RPC:        "rpc3",
			EdgeID:     3,
			CreateTime: 13,
		},
		{
			RPC:        "rpc4",
			EdgeID:     4,
			CreateTime: 14,
		},
		{
			RPC:        "method5",
			EdgeID:     5,
			CreateTime: 15,
		},
	}
	for _, edgeRPC := range edgeRPCs {
		err = dao.CreateEdgeRPC(edgeRPC)
		if err != nil {
			t.Error(err)
		}
	}
	// query on all
	retEdgeRPCs, err := dao.ListEdgeRPCs(&query.EdgeRPCQuery{})
	if err != nil {
		t.Error(err)
	}
	if len(retEdgeRPCs) != 5 {
		t.Error("unmatched length of edge rpcs")
	}

	// query on id
	retEdgeRPCs, err = dao.ListEdgeRPCs(&query.EdgeRPCQuery{
		EdgeID: 4,
	})
	if err != nil {
		t.Error(err)
	}
	if len(retEdgeRPCs) != 1 {
		t.Error("unmatched length of edge rpcs")
	}

	// query on create time
	retEdgeRPCs, err = dao.ListEdgeRPCs(&query.EdgeRPCQuery{
		Query: query.Query{
			StartTime: 11,
			EndTime:   12,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(retEdgeRPCs) != 1 {
		t.Error("unmatched length of edge rpcs")
	}
}
