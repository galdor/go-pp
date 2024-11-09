package pp

import (
	"reflect"
	"time"
)

func ValueString(v reflect.Value) string {
	switch vv := v.Interface().(type) {
	case time.Time:
		return TimeValueString(vv)

	case time.Duration:
		return DurationValueString(vv)
	}

	return ""
}

func TimeValueString(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

func DurationValueString(d time.Duration) string {
	return d.String()
}
