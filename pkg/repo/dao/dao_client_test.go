package dao

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/repo/model"
)

func TestCreateClient(t *testing.T) {
	dao, err := NewDao()
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()
	client := &model.Client{
		ClientID:   0,
		Meta:       "test",
		Addr:       "192.168.1.100",
		CreateTime: time.Now().Unix(),
	}
	err = dao.CreateClient(client)
	if err != nil {
		t.Error(err)
	}
}

// go test -v -bench=. -benchmem
func BenchmarkCreateClient(b *testing.B) {
	dao, err := NewDao()
	if err != nil {
		b.Error(err)
	}
	defer dao.Close()

	index := uint64(0)
	now := time.Now().Unix()
	b.ResetTimer()

	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			new := atomic.AddUint64(&index, 1)
			client := &model.Client{
				ClientID:   new,
				Meta:       "test",
				Addr:       "192.168.1.100",
				CreateTime: now,
			}
			err := dao.CreateClient(client)
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}
