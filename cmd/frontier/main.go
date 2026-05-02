package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/pkg/frontier"
	"k8s.io/klog/v2"
)

// drainSecondsFromEnv 决定 SIGTERM 抵达后、调用 frontier.Close() 前的等待时长。
// 这段时间内进程不接新流量也不主动断已有连接——给上游 kube-proxy
// 把本 pod 从 endpoints 摘除、给已建立的 edge 长连接自然结束的窗口。
//
// 上限由 K8s 侧的 terminationGracePeriodSeconds 控制。Operator 默认设 60s，
// 给 Close 自身留 ~10s 余量，所以 drain 默认 30s 是合理起点。用户可调。
const (
	envDrainSeconds     = "FRONTIER_DRAIN_SECONDS"
	defaultDrainSeconds = 30
)

func drainSecondsFromEnv() int {
	v := os.Getenv(envDrainSeconds)
	if v == "" {
		return defaultDrainSeconds
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		klog.Warningf("invalid %s=%q, falling back to default %ds", envDrainSeconds, v, defaultDrainSeconds)
		return defaultDrainSeconds
	}
	return n
}

func main() {
	frontier, err := frontier.NewFrontier()
	if err != nil {
		klog.Errorf("new frontier err: %s", err)
		return
	}
	frontier.Run()

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())

	if drain := drainSecondsFromEnv(); drain > 0 {
		klog.Infof("frontier received shutdown signal, draining for %ds before close", drain)
		time.Sleep(time.Duration(drain) * time.Second)
	}

	frontier.Close()
}
