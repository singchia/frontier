package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/frontier/test/misc"
	"github.com/singchia/geminio"
	"github.com/singchia/go-timer/v2"
	"github.com/spf13/pflag"
)

var (
	sts     = map[uint64]geminio.Stream{} // streamID stream
	stats   = map[string]uint64{}         // clientID count
	updated bool
	mtx     sync.RWMutex
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30011", "address to dial")
	serviceName := pflag.String("service", "foo", "service name")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	printstats := pflag.Bool("printstats", false, "whether print topic stats")

	pflag.Parse()
	go func() {
		http.ListenAndServe("0.0.0.0:6062", nil)
	}()
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
			fmt.Printf("\033[2K\r stream number now: %d\n", len(sts))
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
		clientID := strconv.FormatUint(st.ClientID(), 10)
		mtx.Lock()
		sts[st.StreamID()] = st
		count, ok := stats[clientID]
		if !ok {
			stats[clientID] = 1
		} else {
			stats[clientID] = count + 1
		}
		updated = true
		mtx.Unlock()

		go func(st geminio.Stream) {
			buf := make([]byte, 128)
			_, err := st.Read(buf)
			if err == io.EOF {
				mtx.Lock()
				delete(sts, st.StreamID())
				updated = true
				mtx.Unlock()
			}
		}(st)
	}
}
