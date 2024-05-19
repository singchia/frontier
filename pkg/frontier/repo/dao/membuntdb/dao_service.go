package membuntdb

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"github.com/tidwall/buntdb"
)

func (dao *dao) ListServices(query *query.ServiceQuery) ([]*model.Service, error) {
	return nil, ErrUnimplemented
}

func (dao *dao) CountServices(query *query.ServiceQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) GetService(serviceID uint64) (*model.Service, error) {
	return nil, ErrUnimplemented
}

func (dao *dao) GetServiceByName(name string) (*model.Service, error) {
	return nil, ErrUnimplemented
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
	serviceRPC := &model.ServiceRPC{}
	err := dao.db.View(func(tx *buntdb.Tx) error {
		var (
			err     error
			finderr error
		)
		pivot := fmt.Sprintf(`{"rpc":"%s"}`, rpc)
		finderr = tx.AscendEqual(IdxServiceRPC_RPC, pivot, func(key, value string) bool {
			// TODO return random one
			err = json.Unmarshal([]byte(value), serviceRPC)
			if err != nil {
				return false
			}
			return false
		})
		if err != nil {
			return err
		}
		if finderr != nil {
			return finderr
		}
		return nil
	})
	return serviceRPC, err
}

func (dao *dao) ListServiceRPCs(query *query.ServiceRPCQuery) ([]string, error) {
	return nil, ErrUnimplemented
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

func (dao *dao) GetServiceTopic(topic string) (*model.ServiceTopic, error) {
	return nil, ErrUnimplemented
}

func (dao *dao) ListServiceTopics(query *query.ServiceTopicQuery) ([]string, error) {
	return nil, ErrUnimplemented
}

func (dao *dao) CountServiceTopics(query *query.ServiceTopicQuery) (int64, error) {
	return 0, ErrUnimplemented
}

func (dao *dao) DeleteServiceTopics(serviceID uint64) error {
	return ErrUnimplemented
}

func (dao *dao) CreateServiceTopic(topic *model.ServiceTopic) error {
	return ErrUnimplemented
}
