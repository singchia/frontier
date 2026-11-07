package repo

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	servicesKeyPrefix      = "frontlas:services:"
	servicesKeyPrefixAll   = "frontlas:services:*"
	servicesAliveKeyPrefix = "frontlas:alive:services:"
	serviceLeasesKeyPrefix = "frontlas:service-leases:"
)

//go:embed lua/service_lease.lua
var serviceLeaseScript string

var (
	ErrServiceOwnerChanged        = errors.New("service heartbeat owner changed")
	ErrServiceRegistrationMissing = errors.New("service registration missing: heartbeat requires address")
)

// serviceCommand 将每个映射的修改、过期清理和聚合在线标记作为一个原子操作。
func (dao *Dao) serviceCommand(ctx context.Context, op string, serviceID uint64, service *Service, expiration time.Duration) *redis.Cmd {
	ttl := expiration.Milliseconds()
	if meta := int64(dao.conf.FrontierManager.Expiration.ServiceMeta) * 1000; meta > 0 && meta < ttl {
		ttl = meta
	}
	data, err := json.Marshal(service)
	if err != nil {
		return redis.NewCmdResult(nil, err)
	}
	return dao.rds.Eval(ctx, serviceLeaseScript,
		[]string{getServiceKey(serviceID), getServiceLeasesKey(serviceID), getAliveServiceKey(serviceID), getFrontierKey(service.FrontierID)},
		op, service.FrontierID, service.Addr, string(data), ttl)
}

func (dao *Dao) writeService(ctx context.Context, op string, serviceID uint64, service *Service, expiration time.Duration) error {
	if service == nil || service.FrontierID == "" || (op != "offline" && expiration.Milliseconds() <= 0) {
		return errors.New("invalid service lease")
	}
	result, err := dao.serviceCommand(ctx, op, serviceID, service, expiration).Int()
	if err != nil {
		return fmt.Errorf("%s service %d: %w", op, serviceID, err)
	}
	switch result {
	case -1:
		return ErrServiceOwnerChanged
	case -2:
		return ErrServiceRegistrationMissing
	case 1:
		return nil
	default:
		return fmt.Errorf("unexpected service lease result: %d", result)
	}
}

// HeartbeatService 补建当前 Frontier 的注册，并只续期自己的租约。
func (dao *Dao) HeartbeatService(ctx context.Context, serviceID uint64, service *Service, expiration time.Duration) error {
	return dao.writeService(ctx, "heartbeat", serviceID, service, expiration)
}

func (dao *Dao) SetServiceAndAlive(serviceID uint64, service *Service, expiration time.Duration) error {
	if service == nil || service.Addr == "" {
		return errors.New("service registration requires address")
	}
	return dao.writeService(context.Background(), "online", serviceID, service, expiration)
}

// DeleteServiceInstance 防止旧连接的 Offline 删除同一 Frontier 上的新连接。
func (dao *Dao) DeleteServiceInstance(ctx context.Context, serviceID uint64, frontierID, addr string) error {
	return dao.writeService(ctx, "offline", serviceID, &Service{FrontierID: frontierID, Addr: addr}, 0)
}

func (dao *Dao) DeleteService(serviceID uint64, frontierID string) error {
	return dao.DeleteServiceInstance(context.Background(), serviceID, frontierID, "")
}

// Deprecated: 必须提供 Frontier 归属，不能替同一服务的所有实例续期。
func (dao *Dao) ExpireService(serviceID uint64, expiration time.Duration) error {
	return errors.New("service heartbeat requires frontier identity; use HeartbeatService")
}

// Deprecated: 使用有租约的 SetServiceAndAlive。
func (dao *Dao) SetService(serviceID uint64, service *Service) error {
	return dao.SetServiceAndAlive(serviceID, service, 30*time.Second)
}

func (dao *Dao) liveServices(ctx context.Context, serviceID uint64) ([]*Service, error) {
	values, err := dao.serviceCommand(ctx, "read", serviceID, &Service{}, 0).StringSlice()
	if err != nil {
		return nil, fmt.Errorf("read service %d: %w", serviceID, err)
	}
	return decodeServices(serviceID, values)
}

func decodeServices(serviceID uint64, values []string) ([]*Service, error) {
	services := make([]*Service, 0, len(values))
	for _, value := range values {
		service := &Service{}
		if err := json.Unmarshal([]byte(value), service); err != nil {
			return nil, fmt.Errorf("decode service %d: %w", serviceID, err)
		}
		services = append(services, service)
	}
	sort.Slice(services, func(i, j int) bool { return services[i].FrontierID < services[j].FrontierID })
	return services, nil
}

// GetService 为单实例 API 选择按 FrontierID 排序的第一个在线映射。
func (dao *Dao) GetService(serviceID uint64) (*Service, error) {
	services, err := dao.liveServices(context.Background(), serviceID)
	if err != nil {
		return nil, err
	}
	if len(services) == 0 {
		return nil, redis.Nil
	}
	return services[0], nil
}

// GetServices 返回指定服务的所有在线 Frontier 映射，缺失服务不产生 nil 项。
func (dao *Dao) GetServices(serviceIDs []uint64) ([]*Service, error) {
	services := []*Service{}
	if len(serviceIDs) == 0 {
		return services, nil
	}
	ctx := context.Background()
	pipe := dao.rds.TxPipeline()
	cmds := make([]*redis.Cmd, 0, len(serviceIDs))
	for _, id := range serviceIDs {
		cmds = append(cmds, pipe.Eval(ctx, serviceLeaseScript,
			[]string{getServiceKey(id), getServiceLeasesKey(id), getAliveServiceKey(id), getFrontierKey("")},
			"read", "", "", "", 0))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("read service leases: %w", err)
	}
	for i, cmd := range cmds {
		values, err := cmd.StringSlice()
		if err != nil {
			return nil, err
		}
		instances, err := decodeServices(serviceIDs[i], values)
		if err != nil {
			return nil, err
		}
		services = append(services, instances...)
	}
	return services, nil
}

type ServiceQuery struct {
	Cursor uint64
	Count  int64
}

func (dao *Dao) GetServicesByCursor(query *ServiceQuery) ([]*Service, uint64, error) {
	keys, cursor, err := dao.rds.Scan(context.Background(), query.Cursor, servicesKeyPrefixAll, query.Count).Result()
	if err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(keys))
	for _, key := range keys {
		id, err := getServiceID(key)
		if err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	services, err := dao.GetServices(ids)
	return services, cursor, err
}

func (dao *Dao) GetAllServiceIDs() ([]uint64, error) {
	ids := []uint64{}
	var cursor uint64
	for {
		keys, next, err := dao.rds.Scan(context.Background(), cursor, servicesKeyPrefixAll, 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			id, err := getServiceID(key)
			if err != nil {
				return nil, err
			}
			live, err := dao.liveServices(context.Background(), id)
			if err != nil {
				return nil, err
			}
			if len(live) > 0 {
				ids = append(ids, id)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return ids, nil
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
	return strconv.ParseUint(strings.TrimPrefix(serviceKey, servicesKeyPrefix), 10, 64)
}
func getAliveServiceKey(serviceID uint64) string {
	return servicesAliveKeyPrefix + strconv.FormatUint(serviceID, 10)
}
func getServiceLeasesKey(serviceID uint64) string {
	return serviceLeasesKeyPrefix + strconv.FormatUint(serviceID, 10)
}
