package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/spf13/pflag"
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30011", "address to dial")
	serviceName := pflag.String("service", "foo", "service name")
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
	svc, err := service.NewService(dialer, opt...)
	if err != nil {
		log.Println("new end err:", err)
		return
	}
	// register
	svc.Register(context.TODO(), "echo", func(ctx context.Context, req geminio.Request, rsp geminio.Response) {
		value := req.Data()
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
