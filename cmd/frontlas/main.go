package main

import (
	"context"
	_ "net/http/pprof"

	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/pkg/frontlas"
	"k8s.io/klog/v2"
)

func main() {
	frontlas, err := frontlas.NewFrontlas()
	if err != nil {
		klog.Errorf("new frontlas err: %s", err)
		return
	}
	frontlas.Run()

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())

	frontlas.Close()
}
