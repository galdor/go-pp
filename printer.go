package pp

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"unsafe"
)

const (
	uintptrSize = unsafe.Sizeof(uintptr(0))
)

var (
	DefaultIndent = "  "
	LinePrefix    = ""
)

type Printer struct {
	Indent     string
	LinePrefix string

	level int

	buf bytes.Buffer
}

func (p *Printer) Print(value any) error {
	return p.PrintTo(value, os.Stdout)
}

func (p *Printer) PrintTo(value any, w io.Writer) error {
	p.init()
	p.printValueLine(value)
	_, err := io.Copy(w, &p.buf)
	return err
}

func (p *Printer) String(value any) string {
	p.init()
	p.printValueLine(value)
	return p.buf.String()
}

func (p *Printer) init() {
	if p.Indent == "" {
		p.Indent = DefaultIndent
	}

	p.buf.Reset()
}

func (p *Printer) printValueLine(value any) {
	p.printLineStart()
	p.printValue(value)
	p.printNewline()
}

func (p *Printer) printValue(value any) {
	var v reflect.Value
	if rv, ok := value.(reflect.Value); ok {
		v = rv
	} else {
		v = reflect.ValueOf(value)
	}

	switch v.Kind() {
	case reflect.Bool:
		p.printBooleanValue(v.Bool())
	case reflect.Int:
		p.printIntegerValue(v.Int())
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.printIntegerValue(v.Int())
	case reflect.Uint:
		p.printUnsignedIntegerValue(v.Uint())
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		p.printUnsignedIntegerValue(v.Uint())
	case reflect.Uintptr:
		p.printPointerAddressValue(uintptr(v.Uint()))
	case reflect.Float32:
		p.printFloatValue(v.Float(), 32)
	case reflect.Float64:
		p.printFloatValue(v.Float(), 64)
	case reflect.Complex64:
		p.printComplexValue(v.Complex(), 64)
	case reflect.Complex128:
		p.printComplexValue(v.Complex(), 128)
	case reflect.String:
		p.printStringValue(v.String())
	case reflect.Array, reflect.Slice:
		p.printSequenceValue(v)
	case reflect.Map:
		p.printMapValue(v)
	case reflect.Struct:
		p.printStructValue(v)
	case reflect.Func:
		p.printFunctionValue(v)
	case reflect.Chan:
		p.printChannelValue(v)
	case reflect.Interface:
		p.printInterfaceValue(v)
	case reflect.Pointer:
		p.printPointerValue(v)
	case reflect.UnsafePointer:
		p.printPointerAddressValue(v.Pointer())
	default:
		p.printUnknownValue(v)
	}
}

func (p *Printer) printLineStart() {
	p.buf.WriteString(p.LinePrefix)

	for range p.level {
		p.buf.WriteString(p.Indent)
	}
}

func (p *Printer) printNewline() {
	fmt.Fprintln(&p.buf)
}

func (p *Printer) printByte(c byte) {
	p.buf.WriteByte(c)
}

func (p *Printer) printBytes(data []byte) {
	p.buf.Write(data)
}

func (p *Printer) printString(s string) {
	p.buf.WriteString(s)
}

func (p *Printer) printFormat(f string, args ...any) {
	fmt.Fprintf(&p.buf, f, args...)
}

func (p *Printer) printBooleanValue(b bool) {
	if b {
		p.printString("true")
	} else {
		p.printString("false")
	}
}

func (p *Printer) printIntegerValue(i int64) {
	s := strconv.FormatInt(i, 10)
	p.printString(s)
}

func (p *Printer) printUnsignedIntegerValue(u uint64) {
	s := strconv.FormatUint(u, 10)
	p.printString(s)
}

func (p *Printer) printFloatValue(f float64, bitSize int) {
	s := strconv.FormatFloat(f, 'f', -1, bitSize)
	p.printString(s)
}

func (p *Printer) printComplexValue(c complex128, bitSize int) {
	// complex64 uses float32 internally, complex128 uses float64
	bitSize /= 2

	rs := strconv.FormatFloat(real(c), 'f', -1, bitSize)
	p.printString(rs)

	is := strconv.FormatFloat(imag(c), 'f', -1, bitSize)
	if is[0] != '+' && is[0] != '-' {
		p.printByte('+')
	}
	p.printString(is)
	p.printByte('i')
}

func (p *Printer) printStringValue(s string) {
	buf := strconv.AppendQuote([]byte{}, s)
	p.printBytes(buf)
}

func (p *Printer) printSequenceValue(v reflect.Value) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		p.printString(v.Type().String())
		p.printString("(nil)")
	} else {
		p.printString(v.Type().String())
		p.printByte('[')
		p.printNewline()
		p.level++

		for i := range v.Len() {
			ev := v.Index(i)

			p.printLineStart()
			p.printValue(ev)
			p.printByte(',')
			p.printNewline()
		}

		p.level--
		p.printLineStart()
		p.printByte(']')
	}
}

func (p *Printer) printMapValue(v reflect.Value) {
	if v.IsNil() {
		p.printString(v.Type().String())
		p.printString("(nil)")
	} else {
		keys := v.MapKeys()

		if len(keys) == 0 {
			p.printString(v.Type().String())
			p.printString("{}")
		} else {
			slices.SortFunc(keys, p.compareMapKeys)

			p.printString(v.Type().String())
			p.printByte('{')
			p.printNewline()
			p.level++

			for _, kv := range keys {
				vv := v.MapIndex(kv)

				p.printLineStart()
				p.printValue(kv)
				p.printString(": ")
				p.printValue(vv)
				p.printByte(',')
				p.printNewline()
			}

			p.level--
			p.printLineStart()
			p.printByte('}')
		}
	}
}

func (p *Printer) compareMapKeys(v1, v2 reflect.Value) int {
	k1 := v1.Kind()
	k2 := v2.Kind()

	if k1 == k2 {
		switch k1 {
		case reflect.Bool:
			b1, b2 := v1.Bool(), v2.Bool()

			if !b1 && b2 {
				return -1
			} else if b1 && !b2 {
				return 1
			}

			return 0

		case reflect.Int:
			fallthrough
		case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			i1, i2 := v1.Int(), v2.Int()

			if i1 < i2 {
				return -1
			} else if i2 < i1 {
				return 1
			}

			return 0

		case reflect.Uint:
			fallthrough
		case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		case reflect.Uintptr:
			u1, u2 := v1.Uint(), v2.Uint()

			if u1 < u2 {
				return -1
			} else if u2 < u1 {
				return 1
			}

			return 0

		case reflect.Float32, reflect.Float64:
			f1, f2 := v1.Float(), v2.Float()

			if f1 < f2 {
				return -1
			} else if f2 < f1 {
				return 1
			}

			return 0

		case reflect.String:
			return strings.Compare(v1.String(), v2.String())

		case reflect.Chan, reflect.Pointer, reflect.UnsafePointer:
			p1, p2 := v1.Pointer(), v2.Pointer()

			if p1 < p2 {
				return -1
			} else if p2 < p1 {
				return 1
			}

			return 0

		default:
			return 0
		}
	}

	return 0
}

func (p *Printer) printStructValue(v reflect.Value) {
	vt := v.Type()
	p.printString(vt.String())

	if vt.NumField() == 0 {
		p.printString("{}")
	} else {
		p.printByte('{')
		p.printNewline()
		p.level++

		for i := range vt.NumField() {
			fv := v.Field(i)
			ft := vt.Field(i)

			p.printLineStart()
			p.printString(ft.Name)
			p.printString(": ")
			p.printValue(fv)
			p.printByte(',')
			p.printNewline()
		}

		p.level--
		p.printLineStart()
		p.printByte('}')
	}
}

func (p *Printer) printChannelValue(v reflect.Value) {
	p.printByte('(')
	p.printString(v.Type().String())
	p.printByte(')')

	p.printByte('(')
	p.printPointerAddressValue(uintptr(v.Pointer()))
	p.printByte(')')
}

func (p *Printer) printFunctionValue(v reflect.Value) {
	p.printByte('(')
	p.printString(v.Type().String())
	p.printByte(')')

	p.printByte('(')
	p.printPointerAddressValue(uintptr(v.Pointer()))
	p.printByte(')')
}

func (p *Printer) printInterfaceValue(v reflect.Value) {
	if v.IsZero() {
		p.printString(v.Type().String())
		p.printString("(nil)")
	} else {
		p.printValue(v.Elem())
	}
}

func (p *Printer) printPointerValue(v reflect.Value) {
	if v.IsZero() {
		p.printString(v.Type().String())
		p.printString("(nil)")
	} else {
		p.printByte('&')
		p.printValue(v.Elem())
	}
}

func (p *Printer) printPointerAddressValue(ptr uintptr) {
	if ptr == 0 {
		p.printString("nil")
	} else {
		p.printString("0x")

		switch uintptrSize {
		case 4:
			p.printFormat("%08x", ptr)
		case 8:
			p.printFormat("%016x", ptr)
		default:
			p.printFormat("%x", ptr)
		}
	}
}

func (p *Printer) printUnknownValue(v reflect.Value) {
	p.printFormat("%#v", v)
}
