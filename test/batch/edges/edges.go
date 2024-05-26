package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/go-timer/v2"
	"github.com/spf13/pflag"
)

var (
	edges = map[int]edge.Edge{}
	mtx   sync.RWMutex
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30012", "address to dial")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	count := pflag.Int("count", 10000, "edges to dial")
	topic := pflag.String("topic", "test", "topic to specific")
	nseconds := pflag.Int("nseconds", 10, "publish message every n seconds for every edge")
	sourceIPs := pflag.String("source_ips", "", "source ips to dial, if your ")
	pprof := pflag.String("pprof", "", "pprof addr to listen")
	pflag.Parse()

	if *pprof != "" {
		go func() {
			http.ListenAndServe(*pprof, nil)
		}()
	}

	ips := []string{}
	idx := 0
	if *sourceIPs != "" {
		ips = strings.Split(*sourceIPs, ",")
		idx = 0

	}

	dialer := func() (net.Conn, error) {
		if len(ips) != 0 {
			for retry := 0; retry < 2; retry++ {
				localAddr := &net.TCPAddr{
					IP: net.ParseIP(ips[idx]),
				}
				dialer := &net.Dialer{
					LocalAddr: localAddr,
					Timeout:   5 * time.Second,
				}
				conn, err := dialer.Dial(*network, *address)
				if err == nil {
					return conn, nil
				}
				if strings.Contains(err.Error(), "cannot assign requested address") ||
					strings.Contains(err.Error(), "address already in use") {
					fmt.Println("source ip:", localAddr.IP.String(), localAddr.Port, err)
					idx += 1
					if idx >= len(ips) {
						return nil, err
					}
					continue
				}
				fmt.Println("source ip:", localAddr.IP.String(), localAddr.Port)
				return nil, err
			}
		}
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

	tmr := timer.NewTimer()
	for i := 0; i < *count; i++ {
		go func(i int) {
			defer wg.Done()
			// avoid congestion of connection
			random := rand.Intn(*count/100) + 1
			time.Sleep(time.Second * time.Duration(random))
			// new edge connection
			cli, err := edge.NewEdge(dialer,
				edge.OptionEdgeLog(armlog.DefaultLog),
				edge.OptionEdgeTimer(tmr),
				edge.OptionServiceBufferSize(128, 128))
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
					fmt.Println("publish err", err)
					break
				}
				time.Sleep(time.Duration(*nseconds) * time.Second)
			}
			mtx.Lock()
			delete(edges, i)
			mtx.Unlock()
			cli.Close()
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
