package edge

import (
	"github.com/jumboframes/armorigo/log"

	"github.com/singchia/go-timer/v2"
)

type Logger log.Logger
type Timer timer.Timer

type edgeOption struct {
	logger Logger
	tmr    Timer
	edgeID *uint64
	meta   []byte
}

type EdgeOption func(*edgeOption)

func OptionEdgeLog(logger Logger) EdgeOption {
	return func(opt *edgeOption) {
		opt.logger = logger
	}
}

// Pre set timer for the sdk
func OptionEdgeTimer(tmr Timer) EdgeOption {
	return func(opt *edgeOption) {
		opt.tmr = tmr
	}
}

// Pre set EdgeID for the sdk
func OptionEdgeID(edgeID uint64) EdgeOption {
	return func(opt *edgeOption) {
		opt.edgeID = &edgeID
	}
}

func OptionEdgeMeta(meta []byte) EdgeOption {
	return func(opt *edgeOption) {
		opt.meta = meta
	}
}
