package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/spf13/pflag"
)

var (
	edges = map[int64]edge.Edge{}
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30012", "address to dial")
	method := pflag.String("method", "echo", "method to specific")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	count := pflag.Int64("count", 10000, "messages to publish")
	concu := pflag.Int64("concu", 10, "concurrency edges to dial")

	pflag.Parse()
	dialer := func() (net.Conn, error) {
		return net.Dial(*network, *address)
	}
	// log
	level, err := armlog.ParseLevel(*loglevel)
	if err != nil {
		fmt.Println("parse log level err:", err)
		return
	}
	armlog.SetLevel(level)
	armlog.SetOutput(os.Stdout)

	// get edges
	for i := int64(0); i < *concu; i++ {
		meta := string(strconv.FormatInt(i, 10))
		cli, err := edge.NewEdge(dialer,
			edge.OptionEdgeLog(armlog.DefaultLog), edge.OptionEdgeMeta([]byte(meta)))
		if err != nil {
			armlog.Info("new edge err:", err)
			return
		}
		edges[i] = cli
	}
	// start to bench
	benchCall(*method, *count)

	// end and collect
	for _, edge := range edges {
		edge.Close()
	}
}

func benchCall(method string, count int64) {
	start := time.Now()

	wg := sync.WaitGroup{}
	wg.Add(len(edges))

	success, failed, index := int64(0), int64(0), int64(0)

	for _, e := range edges {
		go func(edge edge.Edge) {
			defer wg.Done()
			for {
				newindex := atomic.AddInt64(&index, 1)
				if newindex > count {
					break
				}
				data := []byte(strconv.FormatInt(newindex, 10))
				req := edge.NewRequest(data)
				rsp, err := edge.Call(context.TODO(), method, req)
				if err != nil || string(data) != string(rsp.Data()) {
					atomic.AddInt64(&failed, 1)
					continue
				}
				atomic.AddInt64(&success, 1)
			}
		}(e)
	}
	wg.Wait()
	elapse := time.Now().Sub(start)
	fmt.Printf("publish done: %dms, success:%d, failed: %d\n", elapse.Milliseconds(), success, failed)
}
