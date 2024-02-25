package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/v1/edge"
	"github.com/singchia/frontier/api/v1/service"
	"github.com/spf13/pflag"
)

var (
	edges = map[int64]edge.Edge{}
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:2431", "address to dial")
	serviceName := pflag.String("service", "foo", "service name")
	topic := pflag.String("topic", "bench", "topic to specific")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	printmessage := pflag.Bool("printmessage", false, "whether print message out")

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

	// get service
	opt := []service.ServiceOption{service.OptionServiceLog(armlog.DefaultLog), service.OptionServiceName(*serviceName)}
	if *topic != "" {
		opt = append(opt, service.OptionServiceReceiveTopics([]string{*topic}))
	}
	srv, err := service.NewService(dialer, opt...)
	if err != nil {
		log.Println("new end err:", err)
		return
	}
	for {
		msg, err := srv.Receive(context.TODO())
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Println("> receive err:", err)
			fmt.Print(">>> ")
			continue
		}
		msg.Done()
		if *printmessage {
			value := msg.Data()
			fmt.Printf("> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(value))
			fmt.Print(">>> ")
		}
	}
}
