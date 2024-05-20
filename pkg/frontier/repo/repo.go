package repo

import (
	"errors"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/dao/membuntdb"
	"github.com/singchia/frontier/pkg/frontier/repo/dao/memsqlite"
)

func NewRepo(config *config.Configuration) (apis.Repo, error) {
	switch config.Dao.Backend {
	case "", "buntdb":
		return membuntdb.NewDao(config)
	case "sqlite", "sqlite3":
		return memsqlite.NewDao(config)
	}
	return nil, errors.New("unsupported dao backend")
}
