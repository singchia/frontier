package membuntdb

import (
	"fmt"
	"strconv"
	"testing"

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

	count := 1000
	services := genServices(count)
	for _, service := range services {
		err = dao.CreateService(service)
		if err != nil {
			t.Error(err)
		}
	}

	// query on prefix addr
	retServices, err := dao.ListServices(&query.ServiceQuery{
		Addr: "192.168",
		Query: query.Query{
			PageSize: 1000,
			Page:     1,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(retServices) != count {
		t.Error("unmatched length of services", len(retServices))
	}

	// query on prefix addr and create time
	// [12, 13)
	retServices, err = dao.ListServices(&query.ServiceQuery{
		Addr: "192.168",
		Query: query.Query{
			StartTime: 12,
			EndTime:   120,
			PageSize:  200,
			Page:      1,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(retServices) != 108 {
		t.Error("unmatched length of services", len(retServices))
	}

	// query on addr, create time and order
	retServices, err = dao.ListServices(&query.ServiceQuery{
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
	t.Log(len(retServices))
}

func TestDeleteService(t *testing.T) {
	config := &config.Configuration{}
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()

	serviceCount := 100000
	services := genServices(serviceCount)
	for _, service := range services {
		err = dao.CreateService(service)
		if err != nil {
			t.Error(err)
		}
	}

	err = dao.DeleteService(&query.ServiceDelete{
		ServiceID: 1,
	})
	if err != nil {
		t.Error(err)
	}
	services, err = dao.ListServices(&query.ServiceQuery{
		Query: query.Query{
			Page:     1,
			PageSize: serviceCount,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(services) != serviceCount-1 {
		t.Error("unmatched length of services")
	}
}

func TestListServiceRPCs(t *testing.T) {
	config := &config.Configuration{}
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()

	serviceCount := 1000
	rpcPerEdge := 10
	serviceRPCs := genServiceRPCs(serviceCount, rpcPerEdge)
	for _, rpc := range serviceRPCs {
		err = dao.CreateServiceRPC(rpc)
		if err != nil {
			t.Error(err)
		}
	}

	// query on default page
	retServiceRPCs, err := dao.ListServiceRPCs(&query.ServiceRPCQuery{})
	if err != nil {
		t.Error(err)
	}
	if len(retServiceRPCs) != 10 {
		t.Error("unmatched length of service rpcs", len(retServiceRPCs))
	}

	// query on all
	retServiceRPCs, err = dao.ListServiceRPCs(&query.ServiceRPCQuery{
		Query: query.Query{
			Page:     1,
			PageSize: serviceCount * rpcPerEdge,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(retServiceRPCs) != rpcPerEdge {
		t.Error("unmatched length of service rpcs", len(retServiceRPCs))
	}

	// query on id
	retServiceRPCs, err = dao.ListServiceRPCs(&query.ServiceRPCQuery{
		ServiceID: 1,
	})
	if err != nil {
		t.Error(err)
	}
	if len(retServiceRPCs) != rpcPerEdge {
		t.Error("unmatched length of service rpcs", len(retServiceRPCs))
	}

	// query on create time
	retServiceRPCs, err = dao.ListServiceRPCs(&query.ServiceRPCQuery{
		Query: query.Query{
			StartTime: 11,
			EndTime:   12,
		},
	})
	if err != nil {
		t.Error(err)
	}
	if len(retServiceRPCs) != rpcPerEdge {
		t.Error("unmatched length of service rpcs", len(retServiceRPCs))
	}
}

func TestDeleteServiceRPCs(t *testing.T) {
	config := &config.Configuration{}
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()

	serviceCount := 2
	rpcPerEdge := 1000
	serviceRPCs := genServiceRPCs(serviceCount, rpcPerEdge)
	for _, rpc := range serviceRPCs {
		err = dao.CreateServiceRPC(rpc)
		if err != nil {
			t.Error(err)
		}
	}
	err = dao.DeleteServiceRPCs(1)
	if err != nil {
		t.Error(err)
	}
	rpcs, err := dao.ListServiceRPCs(&query.ServiceRPCQuery{
		ServiceID: 1,
	})
	if err != nil {
		t.Error(err)
	}
	if len(rpcs) != 0 {
		t.Error("unmatched length of service rpcs")
	}
}

func genServices(serviceIDCounts int) []*model.Service {
	services := []*model.Service{}
	for i := 0; i < serviceIDCounts; i++ {
		service := &model.Service{
			ServiceID:  uint64(i) + 1,
			Service:    "test" + strconv.Itoa(i+1),
			Addr:       "192.168." + strconv.Itoa(i+1) + ".1",
			CreateTime: int64(i + 1),
		}
		services = append(services, service)
	}
	return services
}

func printServices(services []*model.Service) {
	for _, service := range services {
		fmt.Println(service.ServiceID, service.Service, service.Addr, service.CreateTime)
	}
}

func genServiceRPCs(serviceIDCounts int, rpcPerEdge int) []*model.ServiceRPC {
	serviceRPCs := []*model.ServiceRPC{}
	for i := 0; i < serviceIDCounts; i++ {
		for j := 0; j < rpcPerEdge; j++ {
			serviceRPC := &model.ServiceRPC{
				RPC:        "rpc" + strconv.Itoa(j+1),
				ServiceID:  uint64(i + 1),
				CreateTime: int64(i + 1),
			}
			serviceRPCs = append(serviceRPCs, serviceRPC)
		}
	}
	return serviceRPCs
}
