package repo

import (
	"testing"

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
	dao, err := newDao(config)
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
