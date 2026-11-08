package servicebound

import (
	"context"
	"errors"
	"testing"

	"github.com/singchia/frontier/pkg/frontier/apis"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/geminio"
)

type registrationRepo struct {
	apis.Repo
	err     error
	records []*model.ServiceRPC
}

func (r *registrationRepo) GetServiceRPCs(string) ([]*model.ServiceRPC, error) {
	return r.records, r.err
}

type registrationRequest struct{ geminio.Request }

func (registrationRequest) Data() []byte     { return []byte("callback") }
func (registrationRequest) ClientID() uint64 { return 7 }

type registrationResponse struct {
	geminio.Response
	err  error
	data []byte
}

func (r *registrationResponse) SetError(e error) { r.err = e }
func (r *registrationResponse) SetData(b []byte) { r.data = b }
func TestRegistrationConfirmationRequiresRepositoryVisibility(t *testing.T) {
	repo := &registrationRepo{err: errors.New("write failed")}
	sm := &serviceManager{repo: repo}
	rsp := &registrationResponse{}
	sm.RegistrationStatus(context.Background(), registrationRequest{}, rsp)
	if rsp.err == nil {
		t.Fatal("failed registration reported success")
	}
	repo.err = nil
	repo.records = []*model.ServiceRPC{{ServiceID: 8, RPC: "callback"}}
	rsp = &registrationResponse{}
	sm.RegistrationStatus(context.Background(), registrationRequest{}, rsp)
	if rsp.err == nil {
		t.Fatal("another service registration accepted")
	}
	repo.records = []*model.ServiceRPC{{ServiceID: 7, RPC: "callback"}}
	rsp = &registrationResponse{}
	sm.RegistrationStatus(context.Background(), registrationRequest{}, rsp)
	if rsp.err != nil || string(rsp.data) != "registered" {
		t.Fatal("visible registration not confirmed")
	}
}
