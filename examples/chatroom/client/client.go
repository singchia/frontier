package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/spf13/pflag"
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30012", "address to dial")
	name := pflag.String("name", "alice", "user name to join chatroom")

	pflag.Parse()
	dialer := func() (net.Conn, error) {
		return net.Dial(*network, *address)
	}
	cli, err := edge.NewNoRetryEdge(dialer, edge.OptionEdgeMeta([]byte(*name)))
	if err != nil {
		fmt.Println("new edge err:", err)
		return
	}
	go func() {
		for {
			msg, err := cli.Receive(context.TODO())
			if err == io.EOF {
				return
			}
			if err != nil {
				fmt.Println("\n> receive err:", err)
				fmt.Println(">>> ")
				continue
			}
			msg.Done()
			chat := &Chat{}
			json.Unmarshal(msg.Data(), chat)
			fmt.Printf("\033[2K\r[%10s]: %s\n", chat.User, chat.Msg)
			fmt.Print(">>> ")
		}
	}()

	fmt.Print(">>> ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		msg := cli.NewMessage([]byte(text))
		err = cli.Publish(context.TODO(), "chatroom", msg)
		if err != nil {
			fmt.Printf("publish err: %s", err)
		}
		fmt.Print(">>> ")
	}
}

type Chat struct {
	User string
	Msg  string
}
