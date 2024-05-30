package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio/pkg/id"
	"github.com/spf13/pflag"
)

var (
	clients sync.Map
)

func main() {
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30011", "address to dial")
	pflag.Parse()

	dialer := func() (net.Conn, error) {
		return net.Dial(*network, *address)
	}
	svc, err := service.NewService(dialer, service.OptionServiceReceiveTopics([]string{"chatroom"}))
	if err != nil {
		fmt.Println("new service err:", err)
		return
	}
	err = svc.RegisterGetEdgeID(context.TODO(), getID)
	if err != nil {
		fmt.Println("svc register getID err:", err)
		return
	}
	err = svc.RegisterEdgeOnline(context.TODO(), online)
	if err != nil {
		fmt.Println("svc register online err:", err)
		return
	}
	err = svc.RegisterEdgeOffline(context.TODO(), offline)
	if err != nil {
		fmt.Println("svc register offline err:", err)
		return
	}
	for {
		msg, err := svc.Receive(context.TODO())
		if err == io.EOF {
			return
		}
		msg.Done()

		name := "unknown"
		value, ok := clients.Load(msg.ClientID())
		if ok {
			name = value.(string)
		}

		fmt.Printf("[%10s]: %s\n", name, string(msg.Data()))

		clients.Range(func(key, value any) bool {
			if value.(string) == name {
				return true
			}
			chat := &Chat{
				User: name,
				Msg:  string(msg.Data()),
			}
			data, _ := json.Marshal(chat)
			newmsg := svc.NewMessage(data)
			svc.Publish(context.TODO(), key.(uint64), newmsg)
			return true
		})
	}
}

type Chat struct {
	User string
	Msg  string
}

func getID(meta []byte) (uint64, error) {
	return id.DefaultIncIDCounter.GetID(), nil
}

func online(edgeID uint64, meta []byte, addr net.Addr) error {
	err := error(nil)
	clients.Range(func(key, value any) bool {
		if value.(string) == string(meta) {
			err = errors.New("user exists")
			return false
		}
		return true
	})
	if err != nil {
		fmt.Printf("> online failed: %s, name: %s, addr: %s\n", err, string(meta), addr.String())
		return err
	}
	fmt.Printf("> online, name: %s, addr: %s\n", string(meta), addr.String())
	clients.Store(edgeID, string(meta))
	return err
}

func offline(edgeID uint64, meta []byte, addr net.Addr) error {
	fmt.Printf("> offline, name: %s, addr: %s\n", string(meta), addr.String())
	clients.Delete(edgeID)
	return nil
}
