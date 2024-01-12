package dao

import (
	"github.com/singchia/frontier/pkg/repo/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type Dao struct {
	dbEdge, dbService *gorm.DB
}

func NewDao() (*Dao, error) {
	// we split client and service sqlite3 memory databases, since the concurrent
	// writes perform bad, see https://github.com/mattn/go-sqlite3/issues/274
	dbEdge, err := gorm.Open(sqlite.Open("file:client?mode=memory&cache=shared"))
	if err != nil {
		klog.Errorf("dao open client sqlite3 err: %s", err)
		return nil, err
	}
	sqlDB, err := dbEdge.DB()
	if err != nil {
		klog.Errorf("get client DB err: %s", err)
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	dbService, err := gorm.Open(sqlite.Open("file:service?mode=memory&cache=shared"))
	if err != nil {
		klog.Errorf("dao open service sqlite3 err: %s", err)
		return nil, err
	}
	sqlDB, err = dbService.DB()
	if err != nil {
		klog.Errorf("get service DB err: %s", err)
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	if err = dbEdge.AutoMigrate(&model.Edge{}); err != nil {
		return nil, err
	}
	return &Dao{dbEdge: dbEdge}, nil
}

func (dao *Dao) Close() error {
	sqlDB, err := dao.dbEdge.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
