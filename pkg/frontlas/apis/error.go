package apis

import "errors"

var (
	ErrUnsupportRedisServerMode = errors.New("unsupport redis-server mode")
	ErrExpireFailed             = errors.New("expire failed")
)
