package memsqlite

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
)

func TestListServices(t *testing.T) {
	config := &config.Configuration{}
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()

	// create services
	index := uint64(0)
	now := time.Now().Unix()
	count := 100
	for i := 0; i < count; i++ {
		new := atomic.AddUint64(&index, 1)
		service := &model.Service{
			ServiceID:  new,
			Service:    "test",
			Addr:       "192.168.2.100",
			CreateTime: now,
		}
		err := dao.CreateService(service)
		if err != nil {
			t.Error(err)
			return
		}
	}
	serviceIDs := []uint64{1, 3, 5}
	// create service topics
	for _, serviceID := range serviceIDs {
		rpc := &model.ServiceRPC{
			RPC:        "foo",
			ServiceID:  serviceID,
			CreateTime: now,
		}
		err := dao.CreateServiceRPC(rpc)
		if err != nil {
			t.Error(err)
			return
		}
	}
	// create service rpcs
	for _, serviceID := range serviceIDs {
		topic := &model.ServiceTopic{
			Topic:      "bar",
			ServiceID:  serviceID,
			CreateTime: now,
		}
		err := dao.CreateServiceTopic(topic)
		if err != nil {
			t.Error(err)
			return
		}
	}

	// list
	services, err := dao.ListServices(&query.ServiceQuery{
		RPC:   "foo",
		Topic: "bar",
	})
	if err != nil {
		t.Error(err)
		return
	}
	for _, service := range services {
		t.Log(service.ServiceID)
	}
	if len(services) != 3 {
		t.Error("unmatch services")
		return
	}
}
