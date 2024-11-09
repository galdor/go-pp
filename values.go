package pp

import (
	"reflect"
	"regexp"
	"time"
)

func ValueString(v reflect.Value) string {
	switch vv := v.Interface().(type) {
	case regexp.Regexp:
		return RegexpValueString(&vv)

	case time.Duration:
		return DurationValueString(vv)

	case time.Time:
		return TimeValueString(vv)
	}

	return ""
}

func RegexpValueString(re *regexp.Regexp) string {
	return "/" + re.String() + "/"
}

func DurationValueString(d time.Duration) string {
	return d.String()
}

func TimeValueString(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
