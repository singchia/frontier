package edgebound

import "github.com/singchia/geminio"

func slice2streams(slice []any) []geminio.Stream {
	streams := []geminio.Stream{}
	for _, elem := range slice {
		streams = append(streams, elem.(geminio.Stream))
	}
	return streams
}
