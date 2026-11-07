package repo

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//go:embed lua/edge_heartbeat.lua
var heartbeatEdgeScript string

var (
	ErrEdgeOwnerChanged        = errors.New("edge heartbeat owner changed")
	ErrEdgeRegistrationMissing = errors.New("edge registration missing: heartbeat requires address")
)

// HeartbeatEdge 补建丢失的注册，并为同一归属续期。数量由 FrontierStats 同步，
// 不调用 SetEdgeAndAlive，以免每次心跳都增加 edge_count。
func (dao *Dao) HeartbeatEdge(ctx context.Context, edgeID uint64, edge *Edge, expiration time.Duration) error {
	if edge == nil || edge.FrontierID == "" || expiration.Milliseconds() <= 0 {
		return errors.New("invalid edge heartbeat")
	}
	data, err := json.Marshal(edge)
	if err != nil {
		return fmt.Errorf("encode edge heartbeat: %w", err)
	}
	result, err := dao.rds.Eval(ctx, heartbeatEdgeScript,
		[]string{getEdgeKey(edgeID), getAliveEdgeKey(edgeID)},
		edge.FrontierID, edge.Addr, string(data),
		(time.Duration(dao.conf.FrontierManager.Expiration.EdgeMeta) * time.Second).Milliseconds(),
		expiration.Milliseconds()).Int()
	if err != nil {
		return fmt.Errorf("heartbeat edge %d: %w", edgeID, err)
	}
	switch result {
	case -1:
		return ErrEdgeOwnerChanged
	case -2:
		return ErrEdgeRegistrationMissing
	default:
		return nil
	}
}
