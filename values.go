package pp

import (
	"reflect"
	"regexp"
	"time"
)

func FormatValue(v reflect.Value) string {
	switch vv := v.Interface().(type) {
	case regexp.Regexp:
		return FormatRegexp(&vv)

	case time.Duration:
		return FormatDuration(vv)

	case time.Time:
		return FormatTime(vv)
	}

	return ""
}

func FormatRegexp(re *regexp.Regexp) string {
	return "/" + re.String() + "/"
}

func FormatDuration(d time.Duration) string {
	return d.String()
}

func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}
