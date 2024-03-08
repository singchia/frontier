package repo

import (
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/config"
	"github.com/singchia/frontier/pkg/repo/dao"
)

func NewRepo(config *config.Configuration) (apis.Repo, error) {
	return dao.NewDao(config)
}
