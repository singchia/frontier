package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/spf13/pflag"
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30012", "address to dial")
	name := pflag.String("name", "alice", "user name to join chatroom")
	listen := pflag.String("listen", "127.0.0.1:1935", "rtmp port to proxy")
	pflag.Parse()
	dialer := func() (net.Conn, error) {
		return net.Dial(*network, *address)
	}
	cli, err := edge.NewNoRetryEdge(dialer, edge.OptionEdgeMeta([]byte(*name)))
	if err != nil {
		fmt.Println("new edge err:", err)
		return
	}
	for {
		ln, err := net.Listen("tcp", *listen)
		if err != nil {
			return
		}
		for {
			netconn, err := ln.Accept()
			if err != nil {
				fmt.Printf("accept err: %s\n", err)
				break
			}
			go func() {
				st, err := cli.OpenStream("rtmp")
				if err != nil {
					fmt.Printf("open stream err: %s\n", err)
					return
				}
				wg := new(sync.WaitGroup)
				wg.Add(2)
				go func() {
					defer wg.Done()
					io.Copy(st, netconn)
					netconn.Close()
					st.Close()
				}()
				go func() {
					defer wg.Done()
					io.Copy(netconn, st)
					netconn.Close()
					st.Close()
				}()
				wg.Wait()
			}()
		}
	}
}
