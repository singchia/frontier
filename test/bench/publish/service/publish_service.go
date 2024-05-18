package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/frontier/test/misc"
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
	topic := pflag.String("topic", "bench", "topic to specific")
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
	if *topic != "" {
		opt = append(opt, service.OptionServiceReceiveTopics([]string{*topic}))
	}
	svc, err := service.NewService(dialer, opt...)
	if err != nil {
		log.Println("new end err:", err)
		return
	}
	for {
		msg, err := svc.Receive(context.TODO())
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println("> receive err:", err)
			fmt.Print(">>> ")
			continue
		}
		mtx.Lock()
		count, ok := stats[msg.Topic()]
		if !ok {
			stats[msg.Topic()] = 1
		} else {
			stats[msg.Topic()] = count + 1
		}
		updated = true
		mtx.Unlock()

		msg.Done()
		if *printmessage {
			value := msg.Data()
			fmt.Printf("> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(value))
			fmt.Print(">>> ")
		}
	}
}
