package repo

import (
	"bytes"
	"fmt"
)

// key: serviceID; value: Service
type Service struct {
	Service    string `json:"service"`
	FrontierID string `json:"frontier_id"`
	Addr       string `json:"addr"`
	UpdateTime int64  `json:"update_time"`
}

func (service *Service) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString("{")
	_, err := buffer.WriteString(fmt.Sprintf("service: %s, ", service.Service))
	if err != nil {
		return nil, err
	}
	_, err = buffer.WriteString(fmt.Sprintf("frontierID: %s, ", service.FrontierID))
	if err != nil {
		return nil, err
	}
	_, err = buffer.WriteString(fmt.Sprintf("addr: %s, ", service.Addr))
	if err != nil {
		return nil, err
	}
	_, err = buffer.WriteString(fmt.Sprintf("update_time: %d", service.UpdateTime))
	if err != nil {
		return nil, err
	}
	buffer.WriteString("}")
	return buffer.Bytes(), nil
}
