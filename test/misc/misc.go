package misc

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func PrintMap(m map[string]uint64) {
	var maxLenKey int
	for k := range m {
		if len(k) > maxLenKey {
			maxLenKey = len(k)
		}
	}

	for k, v := range m {
		fmt.Println(k + ": " + strings.Repeat(" ", maxLenKey-len(k)) + strconv.FormatUint(v, 10))
	}
	fmt.Println("    -----", time.Now().Format("2006-01-02 15:04:05"))
}
