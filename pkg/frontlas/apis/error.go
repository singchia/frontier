package apis

import "errors"

var (
	ErrUnsupportRedisServerMode = errors.New("unsupport redis-server mode")
	ErrExpireFailed             = errors.New("expire failed")
	ErrWrongTypeInRedis         = errors.New("wrong type in redis")
	ErrWrongLengthInRedis       = errors.New("wrong length in redis")
)
