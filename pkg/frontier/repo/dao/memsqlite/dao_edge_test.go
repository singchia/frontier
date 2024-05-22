package memsqlite

import (
	"encoding/json"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/singchia/frontier/pkg/frontier/config"
	"github.com/singchia/frontier/pkg/frontier/repo/model"
	"github.com/singchia/frontier/pkg/frontier/repo/query"
)

func TestCreateEdge(t *testing.T) {
	config := &config.Configuration{}
	config.Dao.Backend = "sqlite3"
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()
	client := &model.Edge{
		EdgeID:     0,
		Meta:       "test",
		Addr:       "192.168.1.100",
		CreateTime: time.Now().Unix(),
	}
	err = dao.CreateEdge(client)
	if err != nil {
		t.Error(err)
	}
}

func TestCountEdges(t *testing.T) {
	config := &config.Configuration{}
	config.Dao.Backend = "sqlite3"
	dao, err := NewDao(config)
	if err != nil {
		t.Error(err)
	}
	defer dao.Close()

	index := uint64(0)
	now := time.Now().Unix()
	count := 10000
	for i := 0; i < count; i++ {
		new := atomic.AddUint64(&index, 1)
		client := &model.Edge{
			EdgeID:     new,
			Meta:       "test",
			Addr:       "192.168.1.100",
			CreateTime: now,
		}
		err := dao.CreateEdge(client)
		if err != nil {
			t.Error(err)
			return
		}
	}

	c, err := dao.CountEdges(&query.EdgeQuery{
		Meta: "test",
	})
	if err != nil {
		t.Error(err)
	}
	if c != 10000 {
		t.Error("unmatch count")
	}
}

// go test -v -bench=. -benchmem
func BenchmarkCreateEdge(b *testing.B) {
	config := &config.Configuration{}
	config.Dao.Backend = "sqlite3"
	dao, err := NewDao(config)
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
			client := &model.Edge{
				EdgeID:     new,
				Meta:       "test",
				Addr:       "192.168.1.100",
				CreateTime: now,
			}
			err := dao.CreateEdge(client)
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkGetEdge(b *testing.B) {
	config := &config.Configuration{}
	config.Dao.Backend = "sqlite3"
	dao, err := NewDao(config)
	if err != nil {
		b.Error(err)
	}
	defer dao.Close()

	// insert b.N clients
	index := uint64(0)
	now := time.Now().Unix()
	for i := 0; i < b.N; i++ {
		new := atomic.AddUint64(&index, 1)
		client := &model.Edge{
			EdgeID:     new,
			Meta:       "test",
			Addr:       "192.168.1.100",
			CreateTime: now,
		}
		err := dao.CreateEdge(client)
		if err != nil {
			b.Error(err)
			return
		}
	}
	// get client bench
	b.ResetTimer()
	index = 0
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			new := atomic.AddUint64(&index, 1)
			_, err := dao.GetEdge(new)
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}

func BenchmarkListEdges(b *testing.B) {
	config := &config.Configuration{}
	config.Dao.Backend = "sqlite3"
	dao, err := NewDao(config)
	if err != nil {
		b.Error(err)
	}
	defer dao.Close()

	// insert b.N clients
	index := uint64(0)
	now := time.Now().Unix()
	count := 100000
	for i := 0; i < count; i++ {
		new := atomic.AddUint64(&index, 1)
		client := &model.Edge{
			EdgeID:     new,
			Meta:       "test",
			Addr:       "192.168.1.100",
			CreateTime: now,
		}
		err := dao.CreateEdge(client)
		if err != nil {
			b.Error(err)
			return
		}
	}
	result := map[string]interface{}{}
	// explain first
	tx := dao.dbEdge.
		Raw(`EXPLAIN QUERY PLAN SELECT * FROM clients WHERE meta LIKE "test%" ORDER BY create_time DESC LIMIT 10 OFFSET 570`).
		Scan(&result)
	if tx.Error != nil {
		b.Error(tx.Error)
		return
	}
	data, _ := json.Marshal(result)
	b.Log(string(data))

	// list clients bench
	b.ResetTimer()
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			pageSize := 10
			page := rand.Intn(count/pageSize) + 1
			clients, err := dao.ListEdges(&query.EdgeQuery{
				Meta: "test",
				Query: query.Query{
					PageSize: pageSize,
					Page:     page,
				},
			})
			if err != nil {
				b.Error(err)
				return
			}
			if len(clients) != pageSize {
				b.Error("unmatch number", len(clients))
			}
		}
	})
}

func BenchmarkDeleteEdge(b *testing.B) {
	config := &config.Configuration{}
	config.Dao.Backend = "sqlite3"
	dao, err := NewDao(config)
	if err != nil {
		b.Error(err)
	}
	defer dao.Close()

	// insert b.N clients
	index := uint64(0)
	now := time.Now().Unix()
	for i := 0; i < b.N; i++ {
		new := atomic.AddUint64(&index, 1)
		client := &model.Edge{
			EdgeID:     new,
			Meta:       "test",
			Addr:       "192.168.1.100",
			CreateTime: now,
		}
		err := dao.CreateEdge(client)
		if err != nil {
			b.Error(err)
			return
		}
	}
	// get client bench
	b.ResetTimer()
	index = 0
	b.RunParallel(func(p *testing.PB) {
		for p.Next() {
			new := atomic.AddUint64(&index, 1)
			err := dao.DeleteEdge(&query.EdgeDelete{EdgeID: new})
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}
