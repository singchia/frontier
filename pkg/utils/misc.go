package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"reflect"
	"strings"
)

// TODO test it
func RemoveOmitEmptyTag(obj interface{}) interface{} {
	v := reflect.ValueOf(obj)
	t := reflect.TypeOf(obj)

	if t.Kind() != reflect.Struct {
		return obj
	}

	newStruct := reflect.New(t).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("yaml")

		if tag != "" {
			tags := strings.Split(tag, ",")
			newTags := []string{}
			for _, t := range tags {
				if t != "omitempty" {
					newTags = append(newTags, t)
				}
			}
			newTag := strings.Join(newTags, ",")
			if newTag != "" {
				fieldType.Tag = reflect.StructTag(fmt.Sprintf(`yaml:"%s"`, newTag))
			} else {
				fieldType.Tag = ""
			}
		}
		newStruct.Field(i).Set(field)
	}

	return newStruct.Interface()
}

func IP2Int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func Int2IP(nn uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}
