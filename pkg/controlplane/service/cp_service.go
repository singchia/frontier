package service

import (
	v1 "github.com/singchia/frontier/api/controlplane/v1"
	"github.com/singchia/frontier/pkg/apis"
	"github.com/singchia/frontier/pkg/repo/dao"
)

type ControlPlaneService struct {
	v1.UnimplementedControlPlaneServer

	// dao and repo
	dao          *dao.Dao
	servicebound apis.Servicebound
	edgebound    apis.Edgebound
}

func NewControlPlaneService(dao *dao.Dao, servicebound apis.Servicebound, edgebound apis.Edgebound) *ControlPlaneService {
	cp := &ControlPlaneService{
		dao:          dao,
		servicebound: servicebound,
		edgebound:    edgebound,
	}
	return cp
}
