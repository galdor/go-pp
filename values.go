package pp

import (
	"reflect"
	"time"
)

func ValueString(v reflect.Value) string {
	switch vv := v.Interface().(type) {
	case time.Time:
		return vv.Format(time.RFC3339Nano)
	}

	return ""
}
