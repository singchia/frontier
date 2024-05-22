package memsqlite

import (
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// service
func (dao *dao) ListServices(query *query.ServiceQuery) ([]*model.Service, error) {
	tx := dao.dbService.Model(&model.Service{})
	if dao.config.Dao.Debug {
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

func (dao *dao) CountServices(query *query.ServiceQuery) (int64, error) {
	tx := dao.dbService.Model(&model.Service{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = buildServiceQuery(tx, query)

	var count int64
	tx = tx.Count(&count)
	return count, tx.Error
}

func (dao *dao) GetService(serviceID uint64) (*model.Service, error) {
	tx := dao.dbService.Model(&model.Service{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = tx.Where("service_id = ?", serviceID).Limit(1)

	var service model.Service
	tx = tx.Find(&service)
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &service, tx.Error
}

func (dao *dao) GetServiceByName(name string) (*model.Service, error) {
	tx := dao.dbService.Model(&model.Service{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = tx.Where("service = ?", name).Limit(1)

	var service model.Service
	tx = tx.Find(&service)
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &service, tx.Error
}

type ServiceDelete struct {
	ServiceID uint64
	Addr      string
}

func (dao *dao) DeleteService(delete *query.ServiceDelete) error {
	tx := dao.dbService
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = buildServiceDelete(tx, delete)
	return tx.Delete(&model.Service{}).Error
}

func (dao *dao) CreateService(service *model.Service) error {
	var tx *gorm.DB
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = dao.dbService.Create(service)
	return tx.Error
}

func buildServiceQuery(tx *gorm.DB, query *query.ServiceQuery) *gorm.DB {
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
	return tx
}

func buildServiceDelete(tx *gorm.DB, delete *query.ServiceDelete) *gorm.DB {
	if delete.ServiceID != 0 {
		tx = tx.Where("service_id = ?", delete.ServiceID)
	}
	if delete.Addr != "" {
		tx = tx.Where("addr = ?", delete.Addr)
	}
	return tx
}

// service rpc
// TODO return random one, or by roundrobin strategy
func (dao *dao) GetServiceRPC(rpc string) (*model.ServiceRPC, error) {
	tx := dao.dbService.Model(&model.ServiceRPC{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = tx.Where("rpc = ?", rpc).Limit(1)

	// we not use Fisrt to avoid the warn log when record not found
	// see https://github.com/go-gorm/gorm/issues/4932
	var mrpc model.ServiceRPC
	tx = tx.Find(&mrpc)
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &mrpc, tx.Error
}

func (dao *dao) ListServiceRPCs(query *query.ServiceRPCQuery) ([]string, error) {
	tx := dao.dbService.Model(&model.ServiceRPC{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = buildServiceRPCQuery(tx, query)
	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.PageSize)

	// order
	if query.Order == "" {
		// desc by create_time by default
		query.Order = "service_rpcs.create_time"
		query.Desc = true
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.Order},
		Desc:   query.Desc,
	})

	// find
	rpcs := []string{}
	tx = tx.Distinct("rpc").Find(&rpcs)
	return rpcs, tx.Error
}

func (dao *dao) CountServiceRPCs(query *query.ServiceRPCQuery) (int64, error) {
	tx := dao.dbService.Model(&model.ServiceRPC{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = buildServiceRPCQuery(tx, query)
	// count
	var count int64
	tx = tx.Distinct("rpc").Count(&count)
	return count, tx.Error
}

func (dao *dao) DeleteServiceRPCs(serviceID uint64) error {
	tx := dao.dbService.Where("service_id = ?", serviceID)
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	return tx.Delete(&model.ServiceRPC{}).Error
}

func (dao *dao) CreateServiceRPC(rpc *model.ServiceRPC) error {
	tx := dao.dbService
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	return tx.Create(rpc).Error
}

func buildServiceRPCQuery(tx *gorm.DB, query *query.ServiceRPCQuery) *gorm.DB {
	// join and search
	if query.Service != "" {
		tx = tx.InnerJoins("INNER JOIN services ON services.service_id = service_rpcs.service_id AND service LIKE ?", "%"+query.Service+"%")
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.ServiceID != 0 {
		tx = tx.Where("service_rpcs.service_id = ?", query.ServiceID)
	}
	return tx
}

// service topic

func (dao *dao) GetServiceTopic(topic string) (*model.ServiceTopic, error) {
	tx := dao.dbService.Model(&model.ServiceTopic{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = tx.Where("topic = ?", topic).Limit(1)

	var mtopic model.ServiceTopic
	tx = tx.Find(&mtopic)
	if tx.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return &mtopic, tx.Error
}

func (dao *dao) ListServiceTopics(query *query.ServiceTopicQuery) ([]string, error) {
	tx := dao.dbService.Model(&model.ServiceTopic{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = buildServiceTopicQuery(tx, query)

	// pagination
	if query.Page <= 0 || query.PageSize <= 0 {
		query.Page, query.PageSize = 1, 10
	}
	offset := query.PageSize * (query.Page - 1)
	tx = tx.Offset(offset).Limit(query.PageSize)

	// order
	if query.Order == "" {
		// desc by create_time by default
		query.Order = "service_topics.create_time"
		query.Desc = true
	}
	tx = tx.Order(clause.OrderByColumn{
		Column: clause.Column{Name: query.Order},
		Desc:   query.Desc,
	})

	// find
	topics := []string{}
	tx = tx.Distinct("topic").Find(&topics)
	return topics, tx.Error
}

func (dao *dao) CountServiceTopics(query *query.ServiceTopicQuery) (int64, error) {
	tx := dao.dbService.Model(&model.ServiceTopic{})
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	tx = buildServiceTopicQuery(tx, query)
	// count
	var count int64
	tx = tx.Distinct("topic").Count(&count)
	return count, tx.Error
}

func (dao *dao) DeleteServiceTopics(serviceID uint64) error {
	tx := dao.dbService.Where("service_id = ?", serviceID)
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	return tx.Delete(&model.ServiceTopic{}).Error
}

func (dao *dao) CreateServiceTopic(topic *model.ServiceTopic) error {
	tx := dao.dbService
	if dao.config.Dao.Debug {
		tx = tx.Debug()
	}
	return tx.Create(topic).Error
}

func buildServiceTopicQuery(tx *gorm.DB, query *query.ServiceTopicQuery) *gorm.DB {
	// join and search
	if query.Service != "" {
		tx = tx.InnerJoins("INNER JOIN services ON services.service_id = service_topics.service_id AND service LIKE ?", "%"+query.Service+"%")
	}
	// time range
	if query.StartTime != 0 && query.EndTime != 0 && query.EndTime > query.StartTime {
		tx = tx.Where("create_time >= ? AND create_time < ?", query.StartTime, query.EndTime)
	}
	// equal
	if query.ServiceID != 0 {
		tx = tx.Where("service_topics.service_id = ?", query.ServiceID)
	}
	return tx
}
