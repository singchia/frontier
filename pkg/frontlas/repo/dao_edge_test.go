package repo

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/singchia/frontier/pkg/frontlas/config"
	"github.com/stretchr/testify/assert"
)

func TestSetEdgeAndAlive(t *testing.T) {
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

	edge := &Edge{
		FrontierID: "a",
		Addr:       "192.168.0.1",
		UpdateTime: time.Now().Unix(),
	}
	err = dao.SetEdgeAndAlive(1, edge, 30*time.Second)
	assert.NoError(t, err)

	// get edge
	value, err := s.Get(getEdgeKey(1))
	assert.NoError(t, err)
	retEdge := &Edge{}
	err = json.Unmarshal([]byte(value), retEdge)
	assert.NoError(t, err)
	assert.Equal(t, edge.Addr, retEdge.Addr)
	assert.Equal(t, edge.FrontierID, retEdge.FrontierID)
	assert.Equal(t, edge.UpdateTime, retEdge.UpdateTime)

	// get edge alive
	value, err = s.Get(getAliveEdgeKey(1))
	assert.NoError(t, err)
	_, err = strconv.Atoi(value)
	assert.NoError(t, err)

	// hget frontier edge_count
	value = s.HGet(getFrontierKey("a"), "edge_count")
	count, err := strconv.Atoi(value)
	assert.NoError(t, err)
	assert.Equal(t, count, 1)

	s.FlushAll()
}
