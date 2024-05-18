package misc

import "reflect"

func IsNil(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsNil()
}
