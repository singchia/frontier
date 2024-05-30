package membuntdb

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/misc"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/tidwall/buntdb"
)

func (dao *dao) ListServices(query *query.ServiceQuery) ([]*model.Service, error) {
	var (
		offset, size int
		idx, pivot   string
		desc         bool
		field        int
	)
	if query.RPC != "" {
		field |= 1
		// TODO we must search index rpc first
		return nil, ErrUnsupportedForBuntDB
	}
	if query.Topic != "" {
		field |= 2
		// TODO we must search index topic first
		return nil, ErrUnsupportedForBuntDB
	}
	if query.Addr != "" {
		field |= 4
	}
	if query.Service != "" {
		field |= 8
	}
	if field&(field-1) != 0 {
		// multiple fields set
		return nil, ErrUnsupportedMultipleFieldsForBuntDB
	}

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset = query.PageSize * (query.Page - 1)
	size = query.PageSize

	// order and index
	switch query.Order {
	case "addr":
		if query.Service != "" {
			return nil, ErrUnsupportedForBuntDB
		}
		idx = IdxService_Addr
		desc = query.Desc
		pivot = fmt.Sprintf(`{"addr": "%s"}`, query.Addr)

	case "service":
		if query.Addr != "" {
			return nil, ErrUnsupportedForBuntDB
		}
		idx = IdxService_Service
		desc = query.Desc
		pivot = fmt.Sprintf(`{"service": "%s"}`, query.Service)

	default:
		// desc by create_time by default
		idx = IdxService_CreateTime
		desc = true
	}

	services := []*model.Service{}
	switch idx {
	case IdxService_Addr, IdxService_Service:
		// index on addr or meta
		err := dao.db.View(func(tx *buntdb.Tx) error {
			find := tx.DescendGreaterThan // TODO range [gte, lt)
			if !desc {
				find = tx.AscendGreaterOrEqual
			}
			skip := 0
			finderr := find(idx, pivot, func(key, value string) bool {
				service, keepon, err := serviceMatch(query.Addr, query.Service, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
				if err != nil {
					// TODO
					return true
				}
				if service != nil {
					services = append(services, service)
				}
				return keepon
			})
			return finderr
		})
		return services, err

	default:
		// index on create_time
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.StartTime != 0 && query.EndTime != 0 {
				find := tx.DescendRange
				if desc {
					query.StartTime -= 1
				} else if !desc {
					find = tx.AscendRange
				}
				less := fmt.Sprintf(`{"create_time": %d}`, query.EndTime)
				greater := fmt.Sprintf(`{"create_time": %d}`, query.StartTime)
				skip := 0
				finderr := find(idx, less, greater, func(key, value string) bool {
					service, keepon, err := serviceMatch(query.Addr, query.Service, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if service != nil {
						services = append(services, service)
					}
					return keepon
				})
				return finderr

			} else {
				find := tx.Descend
				if !desc {
					find = tx.Ascend
				}
				skip := 0
				finderr := find(idx, func(key, value string) bool {
					service, keepon, err := serviceMatch(query.Addr, query.Service, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if service != nil {
						services = append(services, service)
					}
					return keepon
				})
				return finderr
			}
		})
		return services, err
	}
}

func serviceMatch(addr string, svc string, startTime int64, endTime int64, offset int, size int, skip *int, desc bool, serviceStr string) (*model.Service, bool, error) {
	service := &model.Service{}
	err := json.Unmarshal([]byte(serviceStr), service)
	if err != nil {
		// TODO
		return nil, true, err
	}
	// pattern unmatch
	if (addr != "" && !strings.HasPrefix(service.Addr, addr)) ||
		(svc != "" && !strings.HasPrefix(service.Service, svc)) {
		if desc {
			// example: test1 test2 test3 test4 foo5, and search on prefix "test" descent, we won't match, but still need to iterate
			return nil, true, nil
		}
		return nil, false, nil
	}
	// time range unmatch
	if startTime != 0 && endTime != 0 && (service.CreateTime < startTime || service.CreateTime >= endTime) {
		// continue
		return nil, true, nil
	}
	// offset and size
	defer func() { *skip = *skip + 1 }()
	if *skip < offset {
		return nil, true, nil
	} else if *skip >= offset+size {
		// break out
		return nil, false, nil
	} else {
		return service, true, nil
	}
}

func (dao *dao) CountServices(query *query.ServiceQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) GetService(serviceID uint64) (*model.Service, error) {
	service := &model.Service{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		value, err := tx.Get(getServiceKey(serviceID))
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(value), service)
		return err
	})
	return service, err
}

func (dao *dao) GetServiceByName(name string) (*model.Service, error) {
	services, err := dao.GetServicesByName(name)
	if err != nil {
		return nil, err
	}
	// random one
	return services[rand.Intn(len(services))], err
}

func (dao *dao) GetServicesByName(name string) ([]*model.Service, error) {
	services := []*model.Service{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		pivot := fmt.Sprintf(`{"service": "%s"}`, name)
		err := tx.AscendEqual(IdxService_Service, pivot, func(key, value string) bool {
			service := &model.Service{}
			err := json.Unmarshal([]byte(value), service)
			if err != nil {
				// TODO shouldn't be here
				return true
			}
			services = append(services, service)
			return true
		})
		return err
	})
	if len(services) == 0 {
		return nil, apis.ErrRecordNotFound
	}
	return services, err
}

func (dao *dao) DeleteService(delete *query.ServiceDelete) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		if delete.ServiceID != 0 {
			_, err := tx.Delete(getServiceKey(delete.ServiceID))
			return err
		}
		return nil
	})
	return err
}

func (dao *dao) CreateService(service *model.Service) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(service)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(getServiceKey(service.ServiceID), string(data), nil)
		return err
	})
	return err
}

func getServiceKey(serviceID uint64) string {
	return "services:" + strconv.FormatUint(serviceID, 10)
}

// service rpc
func (dao *dao) GetServiceRPC(rpc string) (*model.ServiceRPC, error) {
	serviceRPCs, err := dao.GetServiceRPCs(rpc)
	if err != nil {
		return nil, err
	}
	// return random one
	return serviceRPCs[rand.Intn(len(serviceRPCs))], err
}

func (dao *dao) GetServiceRPCs(rpc string) ([]*model.ServiceRPC, error) {
	serviceRPCs := []*model.ServiceRPC{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		pivot := fmt.Sprintf(`{"rpc":"%s"}`, rpc)
		err := tx.AscendEqual(IdxServiceRPC_RPC, pivot, func(key, value string) bool {
			serviceRPC := &model.ServiceRPC{}
			err := json.Unmarshal([]byte(value), serviceRPC)
			if err != nil {
				return true
			}
			serviceRPCs = append(serviceRPCs, serviceRPC)
			return true
		})
		return err
	})
	if len(serviceRPCs) == 0 {
		return nil, apis.ErrRecordNotFound
	}
	return serviceRPCs, err
}

func (dao *dao) ListServiceRPCs(query *query.ServiceRPCQuery) ([]string, error) {
	var (
		offset, size int
		idx, pivot   string
		desc         bool
	)

	if query.Service != "" {
		// TODO we must search index service first
		return nil, ErrUnsupportedForBuntDB
	}

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset = query.PageSize * (query.Page - 1)
	size = query.PageSize

	// order and index
	switch query.Order {
	case "service_id":
		idx = IdxServiceRPC_ServiceID
		desc = query.Desc
		pivot = fmt.Sprintf(`{"service_id": %d}`, query.ServiceID)
	default:
		// desc by create_time by default
		idx = IdxServiceRPC_CreateTime
		desc = true
	}

	rpcs := map[string]struct{}{}
	switch idx {
	case IdxServiceRPC_ServiceID:
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.ServiceID == 0 {
				find := tx.DescendGreaterThan
				if !desc {
					find = tx.AscendGreaterOrEqual
				}
				skip := 0
				finderr := find(idx, pivot, func(key, value string) bool {
					serviceRPC, keepon, err := serviceRPCMatch(0, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if serviceRPC != nil {
						rpcs[serviceRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr
			} else {
				find := tx.DescendEqual
				if !desc {
					find = tx.AscendEqual
				}
				skip := 0
				finderr := find(idx, pivot, func(key, value string) bool {
					serviceRPC, keepon, err := serviceRPCMatch(query.ServiceID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						return true
					}
					if serviceRPC != nil {
						rpcs[serviceRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr
			}
		})
		return misc.GetKeys(rpcs), err

	default:
		// index on create_time
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.StartTime != 0 && query.EndTime != 0 {
				find := tx.DescendRange
				if desc {
					query.StartTime -= 1
				} else if !desc {
					find = tx.AscendRange
				}
				less := fmt.Sprintf(`{"create_time": %d}`, query.EndTime)
				greater := fmt.Sprintf(`{"create_time": %d}`, query.StartTime)
				skip := 0
				finderr := find(idx, less, greater, func(key, value string) bool {
					serviceRPC, keepon, err := serviceRPCMatch(query.ServiceID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if serviceRPC != nil {
						rpcs[serviceRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr

			} else {
				find := tx.Descend
				if !desc {
					find = tx.Ascend
				}
				skip := 0
				finderr := find(idx, func(key, value string) bool {
					serviceRPC, keepon, err := serviceRPCMatch(query.ServiceID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if serviceRPC != nil {
						rpcs[serviceRPC.RPC] = struct{}{}
					}
					return keepon
				})
				return finderr
			}
		})
		return misc.GetKeys(rpcs), err
	}
}

func serviceRPCMatch(serviceID uint64, startTime int64, endTime int64, offset int, size int, skip *int, desc bool, serviceRPCStr string) (*model.ServiceRPC, bool, error) {
	serviceRPC := &model.ServiceRPC{}
	err := json.Unmarshal([]byte(serviceRPCStr), serviceRPC)
	if err != nil {
		// TODO
		return nil, true, err
	}
	if serviceID != 0 && serviceID != serviceRPC.ServiceID {
		// if we use DescendEqual or AscendEqual, it won't hit here
		if desc {
			return nil, true, nil
		}
		return nil, false, nil
	}
	// time range unmatch
	if startTime != 0 && endTime != 0 && (serviceRPC.CreateTime < startTime || serviceRPC.CreateTime >= endTime) {
		// continue
		return nil, true, nil
	}
	// offset and size
	defer func() { *skip = *skip + 1 }()
	if *skip < offset {
		return nil, true, nil
	} else if *skip >= offset+size {
		// break out
		return nil, false, nil
	} else {
		return serviceRPC, true, nil
	}
}

func (dao *dao) CountServiceRPCs(query *query.ServiceRPCQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) DeleteServiceRPCs(serviceID uint64) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		var delkeys []string
		pivot := fmt.Sprintf(`{"service_id":%d}`, serviceID)
		tx.AscendEqual(IdxServiceRPC_ServiceID, pivot, func(key, value string) bool {
			delkeys = append(delkeys, key)
			return true
		})
		for _, key := range delkeys {
			if _, err := tx.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (dao *dao) CreateServiceRPC(rpc *model.ServiceRPC) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(rpc)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(getServiceRPCKey(rpc.ServiceID, rpc.RPC), string(data), nil)
		return err
	})
	return err
}

func getServiceRPCKey(serviceID uint64, rpc string) string {
	return "service_rpcs:" + strconv.FormatUint(serviceID, 10) + "-" + rpc
}

// service topics
func (dao *dao) GetServiceTopic(topic string) (*model.ServiceTopic, error) {
	serviceTopics, err := dao.GetServiceTopics(topic)
	if err != nil {
		return nil, err
	}
	return serviceTopics[rand.Intn(len(serviceTopics))], err
}

func (dao *dao) GetServiceTopics(topic string) ([]*model.ServiceTopic, error) {
	serviceTopics := []*model.ServiceTopic{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		pivot := fmt.Sprintf(`{"topic":"%s"}`, topic)
		err := tx.AscendEqual(IdxServiceTopic_Topic, pivot, func(key, value string) bool {
			serviceTopic := &model.ServiceTopic{}
			err := json.Unmarshal([]byte(value), serviceTopic)
			if err != nil {
				return true
			}
			serviceTopics = append(serviceTopics, serviceTopic)
			return true
		})
		return err
	})
	if len(serviceTopics) == 0 {
		return nil, apis.ErrRecordNotFound
	}
	return serviceTopics, err
}

func (dao *dao) ListServiceTopics(query *query.ServiceTopicQuery) ([]string, error) {
	var (
		offset, size int
		idx, pivot   string
		desc         bool
	)

	if query.Service != "" {
		// TODO we must search index service first
		return nil, ErrUnsupportedForBuntDB
	}

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset = query.PageSize * (query.Page - 1)
	size = query.PageSize

	// order and index
	switch query.Order {
	case "service_id":
		idx = IdxServiceTopic_ServiceID
		desc = query.Desc
		pivot = fmt.Sprintf(`{"service_id": %d}`, query.ServiceID)
	default:
		// desc by create_time by default
		idx = IdxServiceTopic_CreateTime
		desc = true
	}

	rpcs := map[string]struct{}{}
	switch idx {
	case IdxServiceTopic_ServiceID:
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.ServiceID == 0 {
				find := tx.DescendGreaterThan
				if !desc {
					find = tx.AscendGreaterOrEqual
				}
				skip := 0
				finderr := find(idx, pivot, func(key, value string) bool {
					serviceTopic, keepon, err := serviceTopicMatch(0, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if serviceTopic != nil {
						rpcs[serviceTopic.Topic] = struct{}{}
					}
					return keepon
				})
				return finderr
			} else {
				find := tx.DescendEqual
				if !desc {
					find = tx.AscendEqual
				}
				skip := 0
				finderr := find(idx, pivot, func(key, value string) bool {
					serviceTopic, keepon, err := serviceTopicMatch(query.ServiceID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						return true
					}
					if serviceTopic != nil {
						rpcs[serviceTopic.Topic] = struct{}{}
					}
					return keepon
				})
				return finderr
			}
		})
		return misc.GetKeys(rpcs), err

	default:
		// index on create_time
		err := dao.db.View(func(tx *buntdb.Tx) error {
			if query.StartTime != 0 && query.EndTime != 0 {
				find := tx.DescendRange
				if desc {
					query.StartTime -= 1
				} else if !desc {
					find = tx.AscendRange
				}
				less := fmt.Sprintf(`{"create_time": %d}`, query.EndTime)
				greater := fmt.Sprintf(`{"create_time": %d}`, query.StartTime)
				skip := 0
				finderr := find(idx, less, greater, func(key, value string) bool {
					serviceTopic, keepon, err := serviceTopicMatch(query.ServiceID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if serviceTopic != nil {
						rpcs[serviceTopic.Topic] = struct{}{}
					}
					return keepon
				})
				return finderr

			} else {
				find := tx.Descend
				if !desc {
					find = tx.Ascend
				}
				skip := 0
				finderr := find(idx, func(key, value string) bool {
					serviceTopic, keepon, err := serviceTopicMatch(query.ServiceID, query.StartTime, query.EndTime, offset, size, &skip, desc, value)
					if err != nil {
						// TODO
						return true
					}
					if serviceTopic != nil {
						rpcs[serviceTopic.Topic] = struct{}{}
					}
					return keepon
				})
				return finderr
			}
		})
		return misc.GetKeys(rpcs), err
	}
}

func serviceTopicMatch(serviceID uint64, startTime int64, endTime int64, offset int, size int, skip *int, desc bool, serviceTopicStr string) (*model.ServiceTopic, bool, error) {
	serviceTopic := &model.ServiceTopic{}
	err := json.Unmarshal([]byte(serviceTopicStr), serviceTopic)
	if err != nil {
		// TODO
		return nil, true, err
	}
	if serviceID != 0 && serviceID != serviceTopic.ServiceID {
		// if we use DescendEqual or AscendEqual, it won't hit here
		if desc {
			return nil, true, nil
		}
		return nil, false, nil
	}
	// time range unmatch
	if startTime != 0 && endTime != 0 && (serviceTopic.CreateTime < startTime || serviceTopic.CreateTime >= endTime) {
		// continue
		return nil, true, nil
	}
	// offset and size
	defer func() { *skip = *skip + 1 }()
	if *skip < offset {
		return nil, true, nil
	} else if *skip >= offset+size {
		// break out
		return nil, false, nil
	} else {
		return serviceTopic, true, nil
	}
}

func (dao *dao) CountServiceTopics(query *query.ServiceTopicQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) DeleteServiceTopics(serviceID uint64) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		var delkeys []string
		pivot := fmt.Sprintf(`{"service_id":%d}`, serviceID)
		tx.AscendEqual(IdxServiceTopic_ServiceID, pivot, func(key, value string) bool {
			delkeys = append(delkeys, key)
			return true
		})
		for _, key := range delkeys {
			if _, err := tx.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (dao *dao) CreateServiceTopic(topic *model.ServiceTopic) error {
	err := dao.db.Update(func(tx *buntdb.Tx) error {
		data, err := json.Marshal(topic)
		if err != nil {
			return err
		}
		_, _, err = tx.Set(getServiceTopicKey(topic.ServiceID, topic.Topic), string(data), nil)
		return err
	})
	return err
}

func getServiceTopicKey(serviceID uint64, topic string) string {
	return "service_topics:" + strconv.FormatUint(serviceID, 10) + "-" + topic
}
