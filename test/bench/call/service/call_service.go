package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/frontier/test/misc"
	"github.com/singchia/geminio"
	"github.com/singchia/go-timer/v2"
	"github.com/spf13/pflag"
)

var (
	stats   = map[string]uint64{}
	updated bool
	mtx     = sync.RWMutex{}
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30011", "address to dial")
	serviceName := pflag.String("service", "foo", "service name")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	printmessage := pflag.Bool("printmessage", false, "whether print message out")
	printstats := pflag.Bool("printstats", false, "whether print topic stats")

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

	if *printstats {
		t := timer.NewTimer()
		t.Add(10*time.Second, timer.WithCyclically(), timer.WithHandler(func(e *timer.Event) {
			mtx.Lock()
			defer mtx.Unlock()
			if !updated {
				return
			}
			misc.PrintMap(stats)
			updated = false
		}))
	}

	// get service
	opt := []service.ServiceOption{service.OptionServiceLog(armlog.DefaultLog), service.OptionServiceName(*serviceName)}
	svc, err := service.NewService(dialer, opt...)
	if err != nil {
		log.Println("new end err:", err)
		return
	}
	// register
	svc.Register(context.TODO(), "echo", func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		value := req.Data()
		mtx.Lock()
		count, ok := stats[req.Method()]
		if !ok {
			stats[req.Method()] = 1
		} else {
			stats[req.Method()] = count + 1
		}
		updated = true
		mtx.Unlock()
		if *printmessage {
			edgeID := req.ClientID()
			fmt.Printf("\n> call rpc, method: %s edgeID: %d streamID: %d data: %s\n", "echo", edgeID, req.StreamID(), string(value))
			fmt.Print(">>> ")
		}
		rsp.SetData(value)
	})

	sig := sigaction.NewSignal()
	sig.Wait(context.TODO())

	svc.Close()
}
