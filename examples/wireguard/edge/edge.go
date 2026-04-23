// Command wg-edge is the edge-side half of the WireGuard-over-frontier
// example. It listens on a local UDP endpoint for traffic from a WireGuard
// peer, opens one long-lived geminio stream to the wg-router service,
// writes a pair-id first frame, and shuttles UDP datagrams between the
// local peer and the stream.
package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/singchia/frontier/api/dataplane/v1/edge"
	"github.com/singchia/frontier/examples/wireguard/internal/frame"
	"github.com/spf13/pflag"
)

func main() {
	frontierAddr := pflag.String("frontier-addr", "127.0.0.1:30012", "frontier edgebound addr")
	frontierNet := pflag.String("frontier-network", "udp", "tcp | udp")
	listenAddr := pflag.String("listen", "127.0.0.1:51820", "UDP listen addr for local wg peer")
	pairID := pflag.String("pair-id", "hello", "pairing identifier (both sides must match)")
	serviceName := pflag.String("service-name", "wg", "frontier service name for router")
	name := pflag.String("name", "edge", "log prefix name")
	pflag.Parse()

	logger := log.New(os.Stderr, "[wg-edge "+*name+"] ", log.LstdFlags)

	udpAddr, err := net.ResolveUDPAddr("udp", *listenAddr)
	if err != nil {
		logger.Fatalf("resolve listen addr: %v", err)
	}
	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		logger.Fatalf("listen udp: %v", err)
	}
	defer udpConn.Close()
	logger.Printf("listening UDP on %s", udpConn.LocalAddr())

	dialer := func() (net.Conn, error) {
		return net.Dial(*frontierNet, *frontierAddr)
	}
	cli, err := edge.NewEdge(dialer, edge.OptionEdgeMeta([]byte(*name)))
	if err != nil {
		logger.Fatalf("new edge: %v", err)
	}
	defer cli.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go handleSignals(cancel, logger, udpConn)

	var lastSrc atomic.Pointer[net.UDPAddr]

	backoff := time.Second
	for ctx.Err() == nil {
		err := runStream(cli, *serviceName, *pairID, udpConn, &lastSrc, logger)
		if ctx.Err() != nil {
			return
		}
		logger.Printf("stream closed: %v (reconnect in %s)", err, backoff)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

func handleSignals(cancel context.CancelFunc, logger *log.Logger, udpConn *net.UDPConn) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	sig := <-ch
	logger.Printf("signal %s: shutting down", sig)
	cancel()
	udpConn.Close()
}

// runStream runs one stream lifecycle: open, write pair-id, shuttle UDP <-> stream,
// and return when any side errors out. The caller loops to reopen.
func runStream(
	cli edge.Edge,
	serviceName, pairID string,
	udpConn *net.UDPConn,
	lastSrc *atomic.Pointer[net.UDPAddr],
	logger *log.Logger,
) error {
	stream, err := cli.OpenStream(serviceName)
	if err != nil {
		return err
	}
	defer stream.Close()

	if err := frame.WriteFrame(stream, []byte(pairID)); err != nil {
		return err
	}
	logger.Printf("stream opened, pair-id=%q sent", pairID)

	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	// UDP -> stream
	go func() {
		defer wg.Done()
		buf := make([]byte, 65535)
		for {
			n, src, err := udpConn.ReadFromUDP(buf)
			if err != nil {
				errCh <- err
				return
			}
			lastSrc.Store(src)
			if err := frame.WriteFrame(stream, buf[:n]); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// stream -> UDP
	go func() {
		defer wg.Done()
		for {
			pkt, err := frame.ReadFrame(stream)
			if err != nil {
				errCh <- err
				return
			}
			dst := lastSrc.Load()
			if dst == nil {
				// No local peer has spoken yet; drop (WG will retry).
				continue
			}
			if _, err := udpConn.WriteToUDP(pkt, dst); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// Wait for first failure, then close stream to unblock the other goroutine.
	err = <-errCh
	stream.Close()
	wg.Wait()
	// Drain second error without blocking (channel is buffered at 2).
	select {
	case <-errCh:
	default:
	}
	return err
}
