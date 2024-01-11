package dao

import (
	"github.com/singchia/frontier/pkg/repo/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/klog/v2"
)

type Dao struct {
	db *gorm.DB
}

func NewDao() (*Dao, error) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"))
	if err != nil {
		klog.Errorf("client manager open sqlite3 err: %s", err)
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	if err = db.AutoMigrate(&model.Client{}); err != nil {
		return nil, err
	}
	return &Dao{db: db}, nil
}

func (dao *Dao) Close() error {
	sqlDB, err := dao.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
