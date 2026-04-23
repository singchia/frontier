// Command udpping is a minimal UDP send/echo tool used to drive the
// WireGuard-over-frontier example without needing a real WG peer.
package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
)

func main() {
	mode := pflag.String("mode", "send", "send | echo")
	listen := pflag.String("listen", "127.0.0.1:7000", "local UDP addr")
	target := pflag.String("target", "127.0.0.1:51820", "remote UDP addr (send: destination; echo: seed target)")
	interval := pflag.Duration("interval", 1*time.Second, "send interval (send mode only)")
	payload := pflag.String("payload", "ping", "payload to send / seed")
	pflag.Parse()

	logger := log.New(os.Stderr, "[udpping "+*mode+"] ", log.LstdFlags)

	localAddr, err := net.ResolveUDPAddr("udp", *listen)
	if err != nil {
		logger.Fatalf("resolve listen: %v", err)
	}
	remoteAddr, err := net.ResolveUDPAddr("udp", *target)
	if err != nil {
		logger.Fatalf("resolve target: %v", err)
	}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		logger.Fatalf("listen: %v", err)
	}
	defer conn.Close()
	logger.Printf("listening on %s, target %s", conn.LocalAddr(), remoteAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		conn.Close()
	}()

	switch *mode {
	case "send":
		go senderLoop(conn, remoteAddr, *payload, *interval, logger)
		receiveLoop(conn, logger, false)
	case "echo":
		// Seed: one datagram so the far side knows where to reply.
		if _, err := conn.WriteToUDP([]byte("seed"), remoteAddr); err != nil {
			logger.Printf("seed write: %v", err)
		}
		receiveLoop(conn, logger, true)
	default:
		logger.Fatalf("unknown mode %q", *mode)
	}
}

func senderLoop(conn *net.UDPConn, target *net.UDPAddr, payload string, interval time.Duration, logger *log.Logger) {
	t := time.NewTicker(interval)
	defer t.Stop()
	seq := 0
	for range t.C {
		seq++
		msg := fmt.Sprintf("%s #%d", payload, seq)
		if _, err := conn.WriteToUDP([]byte(msg), target); err != nil {
			logger.Printf("write: %v", err)
			return
		}
	}
}

func receiveLoop(conn *net.UDPConn, logger *log.Logger, echo bool) {
	buf := make([]byte, 65535)
	for {
		n, src, err := conn.ReadFromUDP(buf)
		if err != nil {
			logger.Printf("read: %v", err)
			return
		}
		logger.Printf("recv %d bytes from %s: %q", n, src, buf[:n])
		if echo {
			if _, err := conn.WriteToUDP(buf[:n], src); err != nil {
				logger.Printf("echo write: %v", err)
				return
			}
		}
	}
}
