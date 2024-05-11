package repo

import (
	"context"
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"
)

const (
	servicesKeyPrefix    = "frontlas:services:" // example: frontlas:services:a "{}"
	servicesKeyPrefixAll = "frontlas:services:*"

	servicesAliveKeyPrefix = "frontlas:alive:services:" // example: frontlas:alive:services:a 1 ex 30
)

//go:embed lua/service_delete.lua
var deleteFrontierScript string

func (dao *Dao) GetAllServiceIDs() ([]uint64, error) {
	results, err := dao.rds.Keys(context.TODO(), frontiersKeyPrefixAll).Result()
	if err != nil {
		klog.Errorf("dao get all serviceIDs, keys err: %s", err)
		return nil, err
	}
	serviceIDs := []uint64{}
	for _, v := range results {
		serviceID, err := getServiceID(v)
		if err != nil {
			klog.Errorf("dao get all serviceIDs, get serviceID err; %s", err)
			return nil, err
		}
		serviceIDs = append(serviceIDs, serviceID)
	}
	return serviceIDs, nil
}

type ServiceQuery struct {
	Cursor uint64
	Count  int64
}

func (dao *Dao) GetServicesByCursor(query *ServiceQuery) ([]*Service, uint64, error) {
	services := []*Service{}
	keys, cursor, err := dao.rds.Scan(context.TODO(), query.Cursor, frontiersKeyPrefixAll, query.Count).Result()
	if err != nil {
		klog.Errorf("dao get services, scan err: %s", err)
		return nil, 0, err
	}
	if keys == nil || len(keys) == 0 {
		return services, cursor, nil
	}
	results, err := dao.rds.MGet(context.TODO(), keys...).Result()
	if err != nil {
		klog.Errorf("dao get services, mget err: %s, keys: %v", err, keys)
		return nil, 0, err
	}
	for _, elem := range results {
		service := &Service{}
		err = json.Unmarshal([]byte(elem.(string)), service)
		if err != nil {
			klog.Errorf("dao get services, json unmarshal err: %s", err)
			return nil, 0, err
		}
		services = append(services, service)
	}
	return services, cursor, nil
}

func (dao *Dao) GetServices(serviceIDs []uint64) ([]*Service, error) {
	keys := make([]string, len(serviceIDs))
	for i, serviceID := range serviceIDs {
		keys[i] = getServiceKey(serviceID)
	}

	results, err := dao.rds.MGet(context.TODO(), keys...).Result()
	if err != nil {
		klog.Errorf("dao get services, mget err: %s", err)
		return nil, err
	}
	services := []*Service{}
	for i, result := range results {
		if result == nil {
			services[i] = nil
		}
		service := &Service{}
		err = json.Unmarshal([]byte(result.(string)), service)
		if err != nil {
			klog.Errorf("dao get services, json unmarshal err: %s", err)
			return nil, err
		}
	}
	return services, nil
}

func (dao *Dao) GetService(serviceID uint64) (*Service, error) {
	result, err := dao.rds.Get(context.TODO(), getServiceKey(serviceID)).Result()
	if err != nil {
		klog.Errorf("dao get service, get err: %s", err)
		return nil, err
	}
	service := &Service{}
	err = json.Unmarshal([]byte(result), service)
	if err != nil {
		klog.Errorf("dao get service, json unmarshal err: %s", err)
		return nil, err
	}
	return service, nil
}

// obsoleted
func (dao *Dao) SetService(serviceID uint64, service *Service) error {
	data, err := json.Marshal(service)
	if err != nil {
		klog.Errorf("dao set service, json marshal err: %s", err)
		return err
	}
	_, err = dao.rds.Set(context.TODO(), getServiceKey(serviceID), string(data), -1).Result()
	if err != nil {
		klog.Errorf("dao set service, set err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) SetServiceAndAlive(serviceID uint64, service *Service, expiration time.Duration) error {
	serviceData, err := json.Marshal(service)
	if err != nil {
		klog.Errorf("dao set service and alive, json marshal err: %s", err)
		return err
	}

	pipeliner := dao.rds.TxPipeline()
	// service meta TODO expiration to custom
	pipeliner.Set(context.TODO(), getServiceKey(serviceID), serviceData,
		time.Duration(dao.conf.FrontierManager.Expiration.ServiceMeta)*time.Second)
	// alive
	pipeliner.Set(context.TODO(), getAliveServiceKey(serviceID), 1, expiration)
	// frontier service_count
	pipeliner.HIncrBy(context.TODO(), getFrontierKey(service.FrontierID), "service_count", 1)

	_, err = pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao set service and alive, pipeliner exec err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) ExpireService(serviceID uint64, expiration time.Duration) error {
	pipeliner := dao.rds.TxPipeline()
	// service meta TODO expiration to custom
	pipeliner.Expire(context.TODO(), getServiceKey(serviceID),
		time.Duration(dao.conf.FrontierManager.Expiration.ServiceMeta)*time.Second)
	// service alive
	pipeliner.Expire(context.TODO(), getAliveServiceKey(serviceID), expiration)

	cmds, err := pipeliner.Exec(context.TODO())
	if err != nil {
		klog.Errorf("dao expire service, pipeliner err: %s", err)
		return err
	}
	for _, cmd := range cmds {
		if cmd.Err() != nil {
			return cmd.Err()
		}
	}
	return nil
}

func (dao *Dao) DeleteService(serviceID uint64) error {
	_, err := dao.rds.Eval(context.TODO(), deleteFrontierScript,
		[]string{getServiceKey(serviceID), getAliveServiceKey(serviceID), frontiersKeyPrefix}).Result()
	if err != nil {
		klog.Errorf("dao delete service, eval err: %s", err)
		return err
	}
	return nil
}

func (dao *Dao) CountServices() (int, error) {
	frontiers, err := dao.GetAllFrontiers()
	if err != nil {
		return 0, err
	}
	count := 0
	for _, frontier := range frontiers {
		count += frontier.ServiceCount
	}
	return count, nil
}

func getServiceKey(serviceID uint64) string {
	return servicesKeyPrefix + strconv.FormatUint(serviceID, 10)
}

func getServiceID(serviceKey string) (uint64, error) {
	key := strings.TrimPrefix(serviceKey, edgesKeyPrefix)
	return strconv.ParseUint(key, 10, 64)
}

func getAliveServiceKey(serviceID uint64) string {
	return servicesAliveKeyPrefix + strconv.FormatUint(serviceID, 10)
}
