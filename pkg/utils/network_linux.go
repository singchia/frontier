//go:build linux
// +build linux

package utils

import (
	"errors"
	"net"

	"github.com/vishvananda/netlink"
)

func GetDefaultRouteIP(_, _ string) (net.IP, error) {
	addrs, err := getDefaultRoute()
	if err != nil {
		return nil, err
	}
	for _, addr := range addrs {
		switch real := addr.(type) {
		case *net.IPNet:
			return real.IP, nil
		case *net.IPAddr:
			return real.IP, nil
		}
	}
	return nil, errors.New("address not found")
}

func getDefaultRoute() ([]net.Addr, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_ALL)
	if err != nil {
		return nil, err
	}
	defaultRoute := &netlink.Route{}
	for _, route := range routes {
		if route.Dst == nil {
			defaultRoute = &route
			break
		}
	}

	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range interfaces {
		if iface.Index == defaultRoute.LinkIndex {
			return iface.Addrs()
		}
	}
	return nil, errors.New("route index not found")
}
