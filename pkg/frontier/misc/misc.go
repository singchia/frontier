package misc

import "reflect"

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
