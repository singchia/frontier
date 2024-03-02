package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/v1/service"
	"github.com/singchia/geminio"
	"github.com/spf13/pflag"
)

var (
	sts = map[uint64]geminio.Stream{}
	mtx sync.RWMutex
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:2431", "address to dial")
	serviceName := pflag.String("service", "foo", "service name")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")

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

	// service accept streams
	for {
		st, err := svc.AcceptStream()
		if err == io.EOF {
			return
		} else if err != nil {
			// ESC[2K means erase the line
			fmt.Printf("> accept stream err: %s", err)
			continue
		}
		mtx.Lock()
		sts[st.StreamID()] = st
		fmt.Print("\033[2K\r stream number:", len(sts))
		mtx.Unlock()

		go func(st geminio.Stream) {
			buf := make([]byte, 128)
			_, err := st.Read(buf)
			if err == io.EOF {
				mtx.Lock()
				delete(sts, st.StreamID())
				mtx.Unlock()
				fmt.Printf("\033[2K\r stream number: %d", len(sts))
			}
		}(st)
	}
}
