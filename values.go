package pp

import (
	"reflect"
	"regexp"
	"sync/atomic"
	"time"
)

func FormatValue(v reflect.Value) any {
	switch vv := v.Interface().(type) {
	case atomic.Bool:
		return vv.Load()
	case atomic.Int32:
		return vv.Load()
	case atomic.Int64:
		return vv.Load()
	case atomic.Pointer[any]:
		return vv.Load()
	case atomic.Uint32:
		return vv.Load()
	case atomic.Uint64:
		return vv.Load()
	case atomic.Uintptr:
		return vv.Load()
	case atomic.Value:
		return vv.Load()

	case regexp.Regexp:
		return FormatRegexp(&vv)

	case time.Duration:
		return FormatDuration(vv)

	case time.Time:
		return FormatTime(vv)
	}

	return nil
}

func FormatRegexp(re *regexp.Regexp) any {
	return RawString("/" + re.String() + "/")
}

func FormatDuration(d time.Duration) any {
	return RawString(d.String())
}

func FormatTime(t time.Time) any {
	return RawString(t.Format(time.RFC3339Nano))
}
