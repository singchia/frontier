package dao

import (
	"github.com/singchia/frontier/pkg/repo/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServiceQuery struct {
	Query
	// Condition fields
	Service   string
	Addr      string
	RPC       string
	Topic     string
	ServiceID uint64
}

func (dao *Dao) ListServices(query *ServiceQuery) ([]*model.Service, error) {
	tx := dao.dbService.Model(&model.Service{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildServiceQuery(tx, query)

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.PageSize)

	// order
	if query.Order == "" {
		// desc by create_time by default
		query.Order = "services.create_time"
		query.Desc = true
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.Order},
		Desc:   query.Desc,
	})

	// find
	services := []*model.Service{}
	tx = tx.Find(&services)
	return services, tx.Error
}

func (dao *Dao) CountServices(query *ServiceQuery) (int64, error) {
	tx := dao.dbService.Model(&model.Edge{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = buildServiceQuery(tx, query)

	var count int64
	tx = tx.Count(&count)
	return count, tx.Error
}

func (dao *Dao) GetService(serviceID uint64) (*model.Service, error) {
	tx := dao.dbService.Model(&model.Service{})
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = tx.Where("service_id = ?", serviceID)

	var service model.Service
	tx = tx.First(&service)
	return &service, tx.Error
}

func (dao *Dao) DeleteService(serviceID uint64) error {
	tx := dao.dbService.Where("service_id = ?", serviceID)
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	return tx.Delete(&model.Service{}).Error
}

func (dao *Dao) CreateService(service *model.Service) error {
	var tx *gorm.DB
	if dao.config.Log.Verbosity >= 4 {
		tx = tx.Debug()
	}
	tx = dao.dbService.Create(service)
	return tx.Error
}

func buildServiceQuery(tx *gorm.DB, query *ServiceQuery) *gorm.DB {
	// join
	if query.RPC != "" {
		tx = tx.InnerJoins("INNER JOIN service_rpcs ON services.service_id = service_rpcs.service_id AND service_rpcs.rpc = ?", query.RPC)
	}

	if query.Topic != "" {
		tx = tx.InnerJoins("INNER JOIN service_topics ON services.service_id = service_topics.service_id AND service_topics.topic = ?", query.Topic)
	}
	// search
	if query.Service != "" {
		tx = tx.Where("service LIKE ?", query.Service+"%")
	}
	if query.Addr != "" {
		tx = tx.Where("addr LIKE ?", query.Addr)
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.ServiceID != 0 {
		tx = tx.Where("service_id = ?", query.ServiceID)
	}
	return tx
}

// service rpc
func (dao *Dao) CreateServiceRPC(rpc *model.ServiceRPC) error {
	return dao.dbService.Create(rpc).Error
}

// service topic
func (dao *Dao) CreateServiceTopic(rpc *model.ServiceTopic) error {
	return dao.dbService.Create(rpc).Error
}
