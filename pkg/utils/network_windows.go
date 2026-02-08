//go:build windows
// +build windows

package utils

import (
	"errors"
	"net"
)

// GetDefaultRouteIP gets the default route IP by dialing to the target address.
// This is a cross-platform approach that works on Windows.
func GetDefaultRouteIP(network, target string) (net.IP, error) {
	conn, err := net.Dial(network, target)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	addr := conn.LocalAddr()
	switch real := addr.(type) {
	case *net.UDPAddr:
		return real.IP, nil
	case *net.TCPAddr:
		return real.IP, nil
	}
	return nil, errors.New("unsupported address")
}
