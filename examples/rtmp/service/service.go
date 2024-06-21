package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/joy4/av/avutil"
	"github.com/singchia/joy4/av/pktque"
	"github.com/singchia/joy4/av/pubsub"
	"github.com/singchia/joy4/format"
	"github.com/singchia/joy4/format/rtmp"
	"github.com/spf13/pflag"
)

func init() {
	format.RegisterAll()
}

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30011", "address to dial")
	pflag.Parse()

	// service
	dialer := func() (net.Conn, error) {
		return net.Dial(*network, *address)
	}
	svc, err := service.NewService(dialer, service.OptionServiceName("rtmp"))
	if err != nil {
		fmt.Println("new service err:", err)
		return
	}
	// rtmp service
	rtmpserver := &rtmp.Server{}

	l := &sync.RWMutex{}
	type Channel struct {
		que *pubsub.Queue
	}
	channels := map[string]*Channel{}

	rtmpserver.HandlePlay = func(conn *rtmp.Conn) {
		fmt.Println(conn.URL.Path)
		l.RLock()
		ch := channels[conn.URL.Path]
		l.RUnlock()

		if ch != nil {
			cursor := ch.que.Latest()
			filters := pktque.Filters{}

			demuxer := &pktque.FilterDemuxer{
				Filter:  filters,
				Demuxer: cursor,
			}

			avutil.CopyFile(conn, demuxer)
		}
	}
	rtmpserver.HandlePublish = func(conn *rtmp.Conn) {
		l.Lock()
		ch := channels[conn.URL.Path]
		if ch == nil {
			ch = &Channel{}
			ch.que = pubsub.NewQueue()
			channels[conn.URL.Path] = ch
		} else {
			ch = nil
		}
		l.Unlock()
		if ch == nil {
			return
		}

		avutil.CopyFile(ch.que, conn)

		l.Lock()
		delete(channels, conn.URL.Path)
		l.Unlock()
		ch.que.Close()
	}

	rtmpserver.Serve(svc)
}
