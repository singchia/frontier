package misc

import (
	"math/rand"
	"net"
	"reflect"

	"github.com/singchia/frontier/pkg/utils"
)

func IsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}

func GetKeys(m map[string]struct{}) []string {
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func Hash(hashby string, count int, edgeID uint64, addr net.Addr) int {
	switch hashby {
	case "srcip":
		tcpaddr, ok := addr.(*net.TCPAddr)
		if !ok {
			return 0
		}
		ip32 := utils.IP2Int(tcpaddr.IP)
		return int(ip32 % uint32(count))

	case "random":
		return rand.Intn(count)

	default: // "edgeid" or empty
		return int(edgeID % uint64(count))
	}
}
