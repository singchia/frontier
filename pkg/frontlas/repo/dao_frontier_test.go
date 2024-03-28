package repo

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/stretchr/testify/assert"
)

func TestGetAllFrontiers(t *testing.T) {
	s := miniredis.RunT(t)

	config := &config.Configuration{
		Redis: config.Redis{
			Mode: "standalone",
		},
	}
	config.Redis.Standalone.Addr = s.Addr()
	config.Redis.Standalone.Network = "tcp"
	dao, err := NewDao(config)
	assert.NoError(t, err)

	// pre set frontiers
	s.HSet(getFrontierKey("a"), "edge_count", "1")
	s.HSet(getFrontierKey("a"), "advertised_sb_addr", "192.168.0.1")
	s.HSet(getFrontierKey("b"), "edge_count", "0")

	// get frontiers
	frontiers, err := dao.GetAllFrontiers()
	assert.NoError(t, err)
	assert.Equal(t, len(frontiers), 2)
}

func TestSetFrontier(t *testing.T) {
	s := miniredis.RunT(t)

	config := &config.Configuration{
		Redis: config.Redis{
			Mode: "standalone",
		},
	}
	config.Redis.Standalone.Addr = s.Addr()
	config.Redis.Standalone.Network = "tcp"
	dao, err := NewDao(config)
	assert.NoError(t, err)

	// set frontier
	set, err := dao.SetFrontierAndAlive("a", &Frontier{
		AdvertisedServiceboundAddr: "192.168.0.1",
	}, 30*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, set, true)

	addr := s.HGet(getFrontierKey("a"), "advertised_sb_addr")
	assert.Equal(t, addr, "192.168.0.1")

	set, err = dao.SetFrontierAndAlive("a", &Frontier{}, 30*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, set, false)
}
