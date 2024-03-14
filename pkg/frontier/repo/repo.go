package repo

import (
	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/dao"
)

func NewRepo(config *config.Configuration) (apis.Repo, error) {
	return dao.NewDao(config)
}
