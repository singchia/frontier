package utils

import (
	"syscall"
)

func SetRLimit(fileLimit uint64) error {
	var rLimit syscall.Rlimit
	rLimit.Cur = fileLimit
	rLimit.Max = fileLimit
	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	return err
}
