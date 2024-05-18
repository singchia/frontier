package memsqlite

import (
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type dao struct {
	dbEdge, dbService *gorm.DB

	// config
	config config.Configuration
}

func NewDao(config *config.Configuration) (*dao, error) {
	// we split edge and service sqlite3 memory databases, since the concurrent
	// writes perform bad, see https://github.com/mattn/go-sqlite3/issues/274

	// edget bound models
	dbEdge, err := gorm.Open(sqlite.Open("file:edge?mode=memory&cache=shared"))
	if err != nil {
		klog.Errorf("dao open edge sqlite3 err: %s", err)
		return nil, err
	}
	sqlDB, err := dbEdge.DB()
	if err != nil {
		klog.Errorf("get edge DB err: %s", err)
		return nil, err
	}
	sqlDB.Exec("PRAGMA synchronous = OFF;")
	sqlDB.Exec("PRAGMA journal_mode = DELETE;")
	sqlDB.Exec("PRAGMA cache_size = -2000;") // 2MB cache
	sqlDB.Exec("PRAGMA temp_store = MEMORY;")
	sqlDB.Exec("PRAGMA locking_mode = EXCLUSIVE;")
	sqlDB.Exec("PRAGMA mmap_size = 268435456;") // 256MB memory map size
	sqlDB.SetMaxOpenConns(0)
	if err = dbEdge.AutoMigrate(&model.Edge{}, &model.EdgeRPC{}); err != nil {
		return nil, err
	}

	// service bound models
	dbService, err := gorm.Open(sqlite.Open("file:service?mode=memory&cache=shared"), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		klog.Errorf("dao open service sqlite3 err: %s", err)
		return nil, err
	}
	sqlDB, err = dbService.DB()
	if err != nil {
		klog.Errorf("get service DB err: %s", err)
		return nil, err
	}
	sqlDB.Exec("PRAGMA synchronous = OFF;")
	sqlDB.Exec("PRAGMA journal_mode = DELETE;")
	sqlDB.Exec("PRAGMA cache_size = -2000;") // 2MB cache
	sqlDB.Exec("PRAGMA temp_store = MEMORY;")
	sqlDB.Exec("PRAGMA locking_mode = EXCLUSIVE;")
	sqlDB.Exec("PRAGMA mmap_size = 268435456;") // 256MB memory map size
	sqlDB.SetMaxOpenConns(0)
	if err = dbService.AutoMigrate(&model.Service{}, &model.ServiceRPC{}, &model.ServiceTopic{}); err != nil {
		return nil, err
	}
	return &dao{
		dbEdge:    dbEdge,
		dbService: dbService,
	}, nil
}

func (dao *dao) Close() error {
	sqlDB, err := dao.dbEdge.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
