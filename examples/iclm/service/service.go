package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	armlog "github.com/jumboframes/armorigo/log"
	"github.com/jumboframes/armorigo/sigaction"
	"github.com/singchia/frontier/api/dataplane/v1/service"
	"github.com/singchia/geminio"
	"github.com/singchia/geminio/pkg/id"
	"github.com/spf13/pflag"
)

var (
	edgeID uint64
	edges  sync.Map
	sns    sync.Map

	methodSlice  []string
	topicSlice   []string
	printmessage *bool
	srv          service.Service
	sig          *sigaction.Signal
	nostdin      *bool

	labelstats map[string]int64 = map[string]int64{}
	topicstats map[string]int64 = map[string]int64{}
	mtx        sync.RWMutex
)

func addLabelStats(label string, delta int64) {
	mtx.Lock()
	counter, ok := labelstats[label]
	if ok {
		counter += delta
		labelstats[label] = counter
	} else {
		labelstats[label] = delta
	}
	mtx.Unlock()
}

func addTopicStats(topic string, delta int64) {
	mtx.Lock()
	counter, ok := topicstats[topic]
	if ok {
		counter += delta
		topicstats[topic] = counter
	} else {
		topicstats[topic] = delta
	}
	mtx.Unlock()
}

func printLabel() {
	mtx.RLock()
	defer mtx.RUnlock()

	for label, counter := range labelstats {
		fmt.Printf("label: %s, counter: %d\n", label, counter)
	}
}

func printTopic() {
	mtx.RLock()
	defer mtx.RUnlock()

	for topic, counter := range topicstats {
		fmt.Printf("topic: %s, counter: %d\n", topic, counter)
	}
}

type LabelData struct {
	Label string `json:"label"`
	Data  []byte `json:"data"`
}

func main() {
	methodSlice = []string{}

	network := pflag.String("network", "tcp", "network to dial")
	address := pflag.String("address", "127.0.0.1:30011", "address to dial")
	frontlasAddress := pflag.String("frontlas_address", "127.0.0.1:40011", "frontlas address to dial, mutually exclusive with address")
	frontlas := pflag.Bool("frontlas", false, "frontlas or frontier")
	loglevel := pflag.String("loglevel", "info", "log level, trace debug info warn error")
	serviceName := pflag.String("service", "foo", "service name")
	topics := pflag.String("topics", "", "topics to receive message, empty means without consuming")
	topicReceivers := pflag.Int("topic_receivers", 1, "receivers to receive topic messages")
	methods := pflag.String("methods", "", "method name, support echo")
	printmessage = pflag.Bool("printmessage", false, "whether print message out")
	nostdin = pflag.Bool("nostdin", false, "nostdin mode, no stdin will be accepted")
	buffersize := pflag.Int("buffer", 8192, "buffer size set for service")
	stats := pflag.Bool("stats", false, "print statistics or not")

	pflag.Parse()
	go func() {
		http.ListenAndServe("0.0.0.0:6062", nil)
	}()
	// log
	level, err := armlog.ParseLevel(*loglevel)
	if err != nil {
		fmt.Println("parse log level err:", err)
		return
	}
	armlog.SetLevel(level)
	armlog.SetOutput(os.Stdout)

	// get service
	opt := []service.ServiceOption{
		service.OptionServiceLog(armlog.DefaultLog),
		service.OptionServiceName(*serviceName),
		service.OptionServiceBufferSize(*buffersize, *buffersize)}
	if *topics != "" {
		topicSlice = strings.Split(*topics, ",")
		opt = append(opt, service.OptionServiceReceiveTopics(topicSlice))
	}
	if *frontlas {
		srv, err = service.NewClusterService(*frontlasAddress, opt...)
	} else {
		dialer := func() (net.Conn, error) {
			return net.Dial(*network, *address)
		}
		srv, err = service.NewService(dialer, opt...)
	}
	if err != nil {
		log.Println("new end err:", err)
		return
	}
	// pre register methods
	if *methods != "" {
		methodSlice = strings.Split(*methods, ",")
		for _, method := range methodSlice {
			switch method {
			case "echo":
				err = srv.Register(context.TODO(), "echo", echo)
				if err != nil {
					fmt.Println("> register echo err:", err)
					return
				}
			}
		}
	}
	// pre register functions for edges events
	err = srv.RegisterGetEdgeID(context.TODO(), getID)
	if err != nil {
		fmt.Println("> end register getID err:", err)
		return
	}
	err = srv.RegisterEdgeOnline(context.TODO(), online)
	if err != nil {
		fmt.Println("> end register online err:", err)
		return
	}
	err = srv.RegisterEdgeOffline(context.TODO(), offline)
	if err != nil {
		fmt.Println("> end register offline err:", err)
		return
	}

	// label counter
	if *stats {
		go func() {
			ticker := time.NewTicker(time.Second)
			for {
				<-ticker.C
				printLabel()
				printTopic()
			}
		}()
	}

	// service receive
	for i := 0; i < *topicReceivers; i++ {
		go func() {
			for {
				msg, err := srv.Receive(context.TODO())
				if err == io.EOF {
					return
				}
				if err != nil {
					fmt.Println("\n> receive err:", err)
					printPrompt()
					continue
				}
				msg.Done()
				value := msg.Data()
				ld := &LabelData{}
				err = json.Unmarshal(value, ld)
				if err == nil {
					addLabelStats(string(ld.Label), 1)
					value = ld.Data
				}
				addTopicStats(msg.Topic(), 1)
				if *printmessage {
					fmt.Printf("> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(value))
					printPrompt()
				}
			}
		}()
	}

	// service accept streams
	go func() {
		for {
			st, err := srv.AcceptStream()
			if err == io.EOF {
				return
			} else if err != nil {
				fmt.Println("\n> accept stream err:", err)
				continue
			}
			fmt.Println("\n> accept stream", st.ClientID(), st.StreamID())
			printPrompt()
			sns.Store(strconv.FormatUint(st.StreamID(), 10), st)
			go handleStream(st)
		}
	}()

	if !*nostdin {
		cursor := "1"
		printPrompt()

		// the command-line protocol
		// 1. close
		// 2. quit
		// 3. open {edgeID}
		// 4. close {streamID}
		// 5. switch {streamID}
		// 6. publish {msg} #note to switch to stream first
		// 7. publish {edgeID} {msg}
		// 8. call {method} {req} #note to switch to stream first
		// 9. call {edgeID} {method} {req}
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()
			parts := strings.Split(text, " ")
			switch len(parts) {
			case 1:
				if parts[0] == "help" {
					fmt.Println(`the command-line protocol
	1. close
	2. quit
	3. open {edgeID}
	4. close {streamID}
	5. switch {streamID}
	6. publish {msg} #note to switch to stream first
	7. publish {clientId} {msg}
	8. call {method} {req} #note to switch to stream first
	9. call {clientId} {method} {req}`)
					goto NEXT
				}
				// 1. close
				if parts[0] == "close" || parts[0] == "quit" {
					srv.Close()
					goto END
				}
				if parts[0] == "count" {
					count := 0
					edges.Range(func(key, value interface{}) bool {
						count++
						return true
					})
					fmt.Println("> count:", count)
					goto NEXT
				}
			case 2:
				// 1. open {edgeID}
				// 2. close {streamID}
				// 3. switch {streamID}
				// 4. publish {msg}
				if parts[0] == "open" {
					edgeID, err := strconv.ParseUint(parts[1], 10, 64)
					if err != nil {
						fmt.Println("> illegal edgeID", err, parts[1])
						goto NEXT
					}
					// 1. open edgeID
					st, err := srv.OpenStream(context.TODO(), edgeID)
					if err != nil {
						fmt.Println("> open stream err", err)
						goto NEXT
					}
					fmt.Println("> open stream success:", edgeID, st.StreamID())
					sns.Store(strconv.FormatUint(st.StreamID(), 10), st)
					go handleStream(st)
					goto NEXT
				}
				if parts[0] == "close" {
					stream := parts[1]
					sn, ok := sns.LoadAndDelete(stream)
					if !ok {
						fmt.Printf("> stream id: %s not found\n", stream)
						goto NEXT
					}
					sn.(geminio.Stream).Close()
					fmt.Println("> close stream success:", stream)
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
					stream := sn.(geminio.Stream)

					if parts[0] == "publish" {
						msg := stream.NewMessage([]byte(parts[1]))
						err := stream.Publish(context.TODO(), msg)
						if err != nil {
							fmt.Println("> publish err:", err)
							goto NEXT
						}
						fmt.Println("> publish success")
						goto NEXT
					}
				}
			case 3:
				// 1. publish {edgeID} {msg}
				// 2. call {method} {req} if switch to stream
				if cursor != "1" {
					// in stream
					sn, ok := sns.Load(cursor)
					if !ok {
						fmt.Printf("> stream: %s not found\n", cursor)
						goto NEXT
					}
					stream := sn.(geminio.Stream)
					if parts[0] == "call" {
						req := stream.NewRequest([]byte(parts[2]))
						rsp, err := stream.Call(context.TODO(), string(parts[1]), req)
						if err != nil {
							fmt.Println("> call err:", err)
							goto NEXT
						}
						fmt.Println("\n> call success, ret:", string(rsp.Data()))
						goto NEXT
					}
				}
				if parts[0] == "publish" {
					edgeID, err := strconv.ParseUint(parts[1], 10, 64)
					if err != nil {
						fmt.Println("> illegal edge id", err, parts[1])
						goto NEXT
					}
					msg := srv.NewMessage([]byte(parts[2]))
					err = srv.Publish(context.TODO(), edgeID, msg)
					if err != nil {
						fmt.Println("> publish err:", err)
						goto NEXT
					}
					fmt.Println("> publish success")
					goto NEXT
				}
			case 4:
				// call {edgeID} {method} {req}
				if parts[0] == "call" {
					edgeID, err := strconv.ParseUint(parts[1], 10, 64)
					if err != nil {
						log.Print("\n> illegal edge id", err, parts[1])
						goto NEXT
					}
					req := srv.NewRequest([]byte(parts[3]))
					rsp, err := srv.Call(context.TODO(), edgeID, parts[2], req)
					if err != nil {
						log.Print("\n> call err:", err)
						goto NEXT
					}
					fmt.Println("> call success, ret:", string(rsp.Data()))
					goto NEXT
				}
			}
			log.Println("illegal operation")
		NEXT:
			if cursor != "1" {
				fmt.Printf("[%20s] >>> ", cursor)
			} else {
				printPrompt()
			}
		}
	}

	sig = sigaction.NewSignal()
	sig.Wait(context.TODO())
END:
	time.Sleep(10 * time.Second)
}

func handleStream(stream geminio.Stream) {
	go func() {
		for {
			msg, err := stream.Receive(context.TODO())
			if err != nil {
				fmt.Printf("> streamID: %d receive err: %s\n", stream.StreamID(), err)
				printPrompt()
				return
			}
			msg.Done()
			value := msg.Data()
			ld := &LabelData{}
			err = json.Unmarshal(value, ld)
			if err == nil {
				addLabelStats(string(ld.Label), 1)
				value = ld.Data
			}
			if *printmessage {
				fmt.Printf("> receive msg, edgeID: %d streamID: %d data: %s\n", msg.ClientID(), msg.StreamID(), string(value))
				printPrompt()
			}
		}
	}()
	go func() {
		for {
			data := make([]byte, 1024)
			_, err := stream.Read(data)
			if err != nil {
				fmt.Printf("> streamID: %d read err: %s\n", stream.StreamID(), err)
				printPrompt()
				return
			}
			fmt.Println("\n> read data:", stream.ClientID(),
				string(data))
			printPrompt()
		}
	}()
	go func() {
		time.Sleep(200 * time.Millisecond)
		for _, method := range methodSlice {
			switch method {
			case "echo":
				err := stream.Register(context.TODO(), "echo", echo)
				if err != nil {
					fmt.Println("> register echo err:", err)
					return
				}
			}
		}
	}()
}

func snID(edgeID uint64, streamID uint64) string {
	return strconv.FormatUint(edgeID, 10) + "-" + strconv.FormatUint(streamID, 10)
}

func pickedge() uint64 {
	var edgeID uint64
	edges.Range(func(key, value interface{}) bool {
		// TODO 先返回第一个
		edgeID = key.(uint64)
		return false
	})
	return edgeID
}

func getID(meta []byte) (uint64, error) {
	return id.DefaultIncIDCounter.GetID(), nil
}

func online(edgeID uint64, meta []byte, addr net.Addr) error {
	fmt.Printf("> online, edgeID: %d, addr: %s\n", edgeID, addr.String())
	printPrompt()
	edges.Store(edgeID, struct{}{})
	return nil
}

func offline(edgeID uint64, meta []byte, addr net.Addr) error {
	fmt.Printf("> offline, edgeID: %d, addr: %s\n", edgeID, addr.String())
	printPrompt()
	edges.Delete(edgeID)
	return nil
}

func echo(ctx context.Context, req geminio.Request, rsp geminio.Response) {
	value := req.Data()
	ld := &LabelData{}
	err := json.Unmarshal(value, ld)
	if err == nil {
		addLabelStats(string(ld.Label), 1)
		value = ld.Data
	}
	if *printmessage {
		fmt.Printf("> rpc called, method: %s edgeID: %d streamID: %d data: %s\n", "echo", req.ClientID(), req.StreamID(), string(value))
		printPrompt()
	}
	rsp.SetData(value)
}

func printPrompt() {
	if !*nostdin {
		fmt.Print(">>> ")
	}
}
