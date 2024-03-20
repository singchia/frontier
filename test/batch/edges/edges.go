package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/spf13/pflag"
)

var (
	edges = map[int]edge.Edge{}
	mtx   sync.RWMutex
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:2432", "address to dial")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	count := pflag.Int("count", 10000, "messages to publish")
	topic := pflag.String("topic", "test", "topic to specific")
	nseconds := pflag.Int("nseconds", 10, "publish message every n seconds for every edge")

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

	wg := sync.WaitGroup{}
	wg.Add(*count)

	for i := 0; i < *count; i++ {
		go func(i int) {
			defer wg.Done()
			// avoid congestion of connection
			random := rand.Intn(1000) + 1
			time.Sleep(time.Millisecond * time.Duration(random))
			// new edge connection
			cli, err := edge.NewEdge(dialer,
				edge.OptionEdgeLog(armlog.DefaultLog))
			if err != nil {
				armlog.Info("new edge err:", err)
				return
			}
			mtx.Lock()
			edges[i] = cli
			mtx.Unlock()
			// publish message in loop
			for {
				str := strconv.FormatInt(int64(i), 10)
				msg := cli.NewMessage([]byte(str))
				err := cli.Publish(context.TODO(), *topic, msg)
				if err != nil {
					break
				}
				time.Sleep(time.Duration(*nseconds) * time.Second)
			}
			mtx.Lock()
			delete(edges, i)
			mtx.Unlock()
		}(i)
	}

	go func() {
		ticker := time.NewTicker(time.Second * 10)
		for {
			<-ticker.C
			mtx.RLock()
			online := len(edges)
			mtx.RUnlock()
			fmt.Printf("online: %d\n", online)
		}
	}()

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())

	mtx.RLock()
	for _, edge := range edges {
		edge.Close()
	}
	mtx.RUnlock()
}
