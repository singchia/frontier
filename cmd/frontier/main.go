package main

import (
	"context"
	_ "net/http/pprof"

	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/pkg/frontier"
	"k8s.io/klog/v2"
)

func main() {
	frontier, err := frontier.NewFrontier()
	if err != nil {
		klog.Errorf("new frontier err: %s", err)
		return
	}
	frontier.Run()

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())

	frontier.Close()
}
