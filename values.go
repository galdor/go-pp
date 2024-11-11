package pp

import (
	"math/big"
	"reflect"
	"regexp"
	"sync/atomic"
	"time"
	"unsafe"
)

func FormatValue(v reflect.Value) any {
	// If the value is a non-exported variable or field, we will not be able to
	// call Interface() on it. Using the unsafe package allows us to work around
	// it.
	v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()

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

	case big.Int:
		return RawString(vv.String())
	case big.Float:
		return RawString(vv.String())
	case big.Rat:
		return RawString(vv.String())

	case regexp.Regexp:
		return RawString("/" + vv.String() + "/")

	case time.Duration:
		return RawString(vv.String())
	case time.Time:
		return RawString(vv.Format(time.RFC3339Nano))
	}

	return nil
}
