package api

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrEdgeNotOnline    = errors.New("edge not online")
	ErrServiceNotOnline = errors.New("service not online")
	ErrRPCNotOnline     = errors.New("rpc not online")
	ErrTopicNotOnline   = errors.New("topic not online")
	ErrIllegalEdgeID    = errors.New("illegal edgeID")
	ErrRecordNotFound   = gorm.ErrRecordNotFound
)

var (
	ErrStrUseOfClosedConnection = "use of closed network connection"
)
