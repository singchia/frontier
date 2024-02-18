package api

import "errors"

var (
	ErrEdgeNotOnline  = errors.New("edge not online")
	ErrTopicNotOnline = errors.New("topic not online")
)
