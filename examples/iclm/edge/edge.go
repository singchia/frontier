package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/singchia/frontier/api/v1/edge"
	"github.com/spf13/pflag"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/singchia/geminio"
)

var (
	sns         sync.Map
	methodSlice []string
)

type LabelData struct {
	Label string `json:"label"`
	Data  []byte `json:"data"`
}

func main() {
	methodSlice = []string{}
	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:2432", "address to dial")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	meta := pflag.String("meta", "test", "meta to set on connection")
	methods := pflag.String("methods", "", "method name, support echo, calculate")
	label := pflag.String("label", "label-01", "label to message or rpc")

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

	// get edge
	cli, err := edge.NewEdge(dialer,
		edge.OptionEdgeLog(armlog.DefaultLog), edge.OptionEdgeMeta([]byte(*meta)))
	if err != nil {
		armlog.Info("new edge err:", err)
		return
	}
	//sms := cli.ListStreams()
	//sns.Store("1", sms[0])
	if *methods != "" {
		methodSlice = strings.Split(*methods, ",")
	}

	// receive on edge
	go func() {
		for {
			msg, err := cli.Receive(context.TODO())
			if err == io.EOF {
				return
			}
			if err != nil {
				fmt.Println("> receive err:", err)
				fmt.Println(">>> ")
				continue
			}
			msg.Done()
			fmt.Printf("\n> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(msg.Data()))
			fmt.Print(">>> ")
		}
	}()

	go func() {
		for {
			st, err := cli.AcceptStream()
			if err == io.EOF {
				return
			} else if err != nil {
				fmt.Println("> accept stream err:", err)
				fmt.Print(">>> ")
				continue
			}
			fmt.Println("> accept stream", st.StreamID())
			fmt.Print(">>> ")
			sns.Store(strconv.FormatUint(st.StreamID(), 10), st)
			go handleStream(st)
		}
	}()

	go func() {
		time.Sleep(200 * time.Millisecond)
		// register
		for _, method := range methodSlice {
			switch method {
			case "echo":
				err = cli.Register(context.TODO(), "echo", echo)
				if err != nil {
					armlog.Info("> register echo err:", err)
					return
				}
			}
		}
	}()

	cursor := "1"
	fmt.Print(">>> ")

	// the command-line protocol
	// 1. close
	// 2. quit
	// 3. switch {streamID}
	// 4. open {service}
	// 5. close {streamID}
	// 6. publish {msg} #note to switch to stream first
	// 7. publish {topic} {msg}
	// 8. call {method} {req}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := scanner.Text()
		parts := strings.Split(text, " ")
		switch len(parts) {
		case 1:
			// 1. close
			// 2. quit
			if parts[0] == "help" {
				fmt.Println(`the cli protocol
	1. close
	2. quit
	3. open {service}
	4. close {streamID}
	5. switch {streamID}
	6. publish {msg} #note to switch to stream first
	7. publish {topic} {msg}
	8. call {method} {req}`)
				goto NEXT
			}
			if parts[0] == "quit" || parts[0] == "close" {
				cli.Close()
				goto END
			}
		case 2:
			// 1. open {service}
			// 2. close {streamID}
			// 3. switch {streamID}
			// 4. publish {msg}
			if parts[0] == "open" {
				service := parts[1]
				st, err := cli.OpenStream(service)
				if err != nil {
					fmt.Println("> open stream err:", err)
					goto NEXT
				}
				fmt.Println("> open stream success:", st.StreamID())
				sns.Store(strconv.FormatUint(st.StreamID(), 10), st)
				go handleStream(st)
				goto NEXT
			}
			if parts[0] == "close" {
				// close sessionID
				session := parts[1]
				sn, ok := sns.LoadAndDelete(session)
				if !ok {
					fmt.Printf("> stream id: %s not found\n", session)
					goto NEXT
				}
				sn.(geminio.Stream).Close()
				fmt.Println("> close stream success:", session)
				goto NEXT
			}
			if parts[0] == "switch" {
				session := parts[1]
				if session == "1" {
					cursor = session
					fmt.Println("> swith stream success:", session)
					goto NEXT
				}
				_, ok := sns.Load(session)
				if !ok {
					fmt.Println("> swith stream failed, not found:", session)
					goto NEXT
				}
				cursor = session
				fmt.Println("> swith stream success:", session)
				goto NEXT
			}
			if cursor != "1" && (parts[0] == "publish") {
				sn, ok := sns.Load(cursor)
				if !ok {
					fmt.Printf("> stream: %s not found\n", cursor)
					goto NEXT
				}

				if parts[0] == "publish" {
					ld := &LabelData{
						Label: *label,
						Data:  []byte(parts[1]),
					}
					data, _ := json.Marshal(ld)
					msg := cli.NewMessage(data)
					err := sn.(geminio.Stream).Publish(context.TODO(), msg)
					if err != nil {
						fmt.Println("> publish err:", err)
						goto NEXT
					}
					fmt.Println("> publish success")
					goto NEXT
				}
			}
		case 3:
			// 1. publish {topic} {msg}
			// 2. call {method} {req}
			if cursor != "1" {
				// in stream
				sn, ok := sns.Load(cursor)
				if !ok {
					fmt.Printf("> stream: %s not found\n", cursor)
					goto NEXT
				}
				if parts[0] == "call" {
					req := cli.NewRequest([]byte(parts[2]))
					rsp, err := sn.(geminio.Stream).Call(context.TODO(), string(parts[1]), req)
					if err != nil {
						fmt.Println("> call err:", err)
						goto NEXT
					}
					fmt.Println("> call success, ret:", string(rsp.Data()))
					goto NEXT
				}
			}
			if parts[0] == "publish" {
				ld := &LabelData{
					Label: *label,
					Data:  []byte(parts[2]),
				}
				data, _ := json.Marshal(ld)
				msg := cli.NewMessage(data)
				err := cli.Publish(context.TODO(), string(parts[1]), msg)
				if err != nil {
					fmt.Println("> publish err:", err)
					goto NEXT
				}
				fmt.Println("> publish success")
				goto NEXT
			}
			if parts[0] == "call" {
				ld := &LabelData{
					Label: *label,
					Data:  []byte(parts[2]),
				}
				data, _ := json.Marshal(ld)
				req := cli.NewRequest(data)
				rsp, err := cli.Call(context.TODO(), string(parts[1]), req)
				if err != nil {
					fmt.Println("> call err:", err)
					goto NEXT
				}
				fmt.Println("> call success, ret:", string(rsp.Data()))
				goto NEXT
			}
		}
		fmt.Println("> illegal operation")
	NEXT:
		if cursor != "1" {
			fmt.Printf("[%20s] >>> ", cursor)
		} else {
			fmt.Print(">>> ")
		}
	}
END:
	time.Sleep(time.Second)
}

func handleStream(stream geminio.Stream) {
	go func() {
		for {
			msg, err := stream.Receive(context.TODO())
			if err != nil {
				fmt.Println("> receive err:", err)
				fmt.Print(">>> ")
				return
			}
			msg.Done()
			fmt.Printf("\n> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(msg.Data()))
			fmt.Print(">>> ")
		}
	}()
	go func() {
		for {
			data := make([]byte, 1024)
			_, err := stream.Read(data)
			if err != nil {
				fmt.Println("> read err:", err)
				fmt.Print(">>> ")
				return
			}
			fmt.Println("> read data:", stream.ClientID(),
				string(data))
			fmt.Print(">>> ")
		}
	}()
	go func() {
		time.Sleep(200 * time.Millisecond)
		for _, method := range methodSlice {
			switch method {
			case "echo":
				err := stream.Register(context.TODO(), "echo", echo)
				if err != nil {
					armlog.Info("> register echo err:", err)
					return
				}
			}
		}
	}()
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	edgeID := req.ClientID()
	fmt.Printf("\n> call rpc, method: %s edgeID: %d streamID: %d data: %s\n", "echo", edgeID, req.StreamID(), string(req.Data()))
	fmt.Print(">>> ")
	rsp.SetData(req.Data())
}
