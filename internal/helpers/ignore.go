package helpers

import (
	"reflect"
	"time"
)

func Ignore(typ reflect.Type) bool {
	return typ == reflect.TypeOf(time.Time{})
}
