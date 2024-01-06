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
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		klog.Errorf("client manager open sqlite3 err: %s", err)
		return nil, err
	}
	if err = db.AutoMigrate(&model.Client{}); err != nil {
		return nil, err
	}
	return &Dao{db: db}, nil
}
