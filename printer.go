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

type PrintTypes string

const (
	PrintTypesDefault PrintTypes = "default"
	PrintTypesAlways  PrintTypes = "always"
	PrintTypesNever   PrintTypes = "never"
)

const (
	uintptrSize = unsafe.Sizeof(uintptr(0))
)

var (
	DefaultIndent = "  "
)

type Printer struct {
	Indent     string
	LinePrefix string
	PrintTypes PrintTypes

	level int

	cyclic   bool
	pointers map[uintptr]*pointerRef

	buf []byte
}

type pointerRef struct {
	n         int
	idx       int
	annotated bool
}

func (p *Printer) Print(value any, label ...any) error {
	return p.PrintTo(os.Stdout, value, label...)
}

func (p *Printer) PrintTo(w io.Writer, value any, label ...any) error {
	p.reset()
	p.maybePrintLabel(label...)
	p.printValueLine(value)
	_, err := w.Write(p.buf)
	return err
}

func (p *Printer) String(value any, label ...any) string {
	p.reset()
	p.maybePrintLabel(label...)
	p.printValueLine(value)
	return string(p.buf)
}

func (p *Printer) reset() {
	if p.Indent == "" {
		p.Indent = DefaultIndent
	}

	if p.PrintTypes == "" {
		p.PrintTypes = PrintTypesDefault
	}

	p.cyclic = false
	p.pointers = make(map[uintptr]*pointerRef)

	p.buf = nil
}

func (p *Printer) storePointer(ptr uintptr) {
	ref := pointerRef{
		n:   len(p.pointers) + 1,
		idx: len(p.buf),
	}

	p.pointers[ptr] = &ref
}

func (p *Printer) annotatePointer(ref *pointerRef) {
	if !ref.annotated {
		before, after := p.buf[:ref.idx], p.buf[ref.idx:]

		s := fmt.Sprintf("#%d=", ref.n)
		p.buf = bytes.Join([][]byte{before, after}, []byte(s))

		ref.annotated = true
	}
}

func (p *Printer) checkPointer(ptr uintptr) bool {
	if ref, found := p.pointers[ptr]; found {
		p.cyclic = true
		p.annotatePointer(ref)

		p.printFormat("#%d#", ref.n)
		return false
	}

	p.storePointer(ptr)
	return true
}

func (p *Printer) maybePrintLabel(label ...any) {
	if len(label) > 0 {
		format, ok := label[0].(string)
		if !ok {
			panic("label format is not a string")
		}

		p.printLineStart()
		p.printFormat(format, label[1:]...)
		p.printByte(':')
		p.printNewline()
	}
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
		p.printBooleanValue(v)
	case reflect.Int:
		fallthrough
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p.printIntegerValue(v)
	case reflect.Uint:
		fallthrough
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		p.printUnsignedIntegerValue(v)
	case reflect.Uintptr:
		p.printPointerAddressValue(uintptr(v.Uint()))
	case reflect.Float32:
		p.printFloatValue(v, 32)
	case reflect.Float64:
		p.printFloatValue(v, 64)
	case reflect.Complex64:
		p.printComplexValue(v, 64)
	case reflect.Complex128:
		p.printComplexValue(v, 128)
	case reflect.String:
		p.printStringValue(v)
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
	p.printString(p.LinePrefix)

	for range p.level {
		p.printString(p.Indent)
	}
}

func (p *Printer) printNewline() {
	p.printByte('\n')
}

func (p *Printer) printByte(c byte) {
	p.buf = append(p.buf, c)
}

func (p *Printer) printBytes(data []byte) {
	p.buf = append(p.buf, data...)
}

func (p *Printer) printString(s string) {
	p.printBytes([]byte(s))
}

func (p *Printer) printFormat(format string, args ...any) {
	p.printString(fmt.Sprintf(format, args...))
}

func (p *Printer) printBooleanValue(v reflect.Value) {
	if p.PrintTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	if b := v.Bool(); b {
		p.printString("true")
	} else {
		p.printString("false")
	}

	if p.PrintTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printIntegerValue(v reflect.Value) {
	if p.PrintTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	i := v.Int()
	s := strconv.FormatInt(i, 10)
	p.printString(s)

	if p.PrintTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printUnsignedIntegerValue(v reflect.Value) {
	if p.PrintTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	u := v.Uint()
	s := strconv.FormatUint(u, 10)
	p.printString(s)

	if p.PrintTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printFloatValue(v reflect.Value, bitSize int) {
	if p.PrintTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	f := v.Float()
	s := strconv.FormatFloat(f, 'f', -1, bitSize)
	p.printString(s)

	if p.PrintTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printComplexValue(v reflect.Value, bitSize int) {
	if p.PrintTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	c := v.Complex()

	bitSize /= 2 // complex64 uses float32 internally, complex128 uses float64

	rs := strconv.FormatFloat(real(c), 'f', -1, bitSize)
	p.printString(rs)

	is := strconv.FormatFloat(imag(c), 'f', -1, bitSize)
	if is[0] != '+' && is[0] != '-' {
		p.printByte('+')
	}
	p.printString(is)
	p.printByte('i')

	if p.PrintTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printStringValue(v reflect.Value) {
	if p.PrintTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	s := v.String()
	buf := strconv.AppendQuote([]byte{}, s)
	p.printBytes(buf)

	if p.PrintTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printSequenceValue(v reflect.Value) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		if p.PrintTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.PrintTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		if v.Kind() == reflect.Slice && !p.checkPointer(v.Pointer()) {
			return
		}

		if p.PrintTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
		}

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
		if p.PrintTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.PrintTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		if !p.checkPointer(v.Pointer()) {
			return
		}

		keys := v.MapKeys()

		if len(keys) == 0 {
			if p.PrintTypes != PrintTypesNever {
				p.printString(p.valueTypeString(v))
			}

			p.printString("{}")
		} else {
			slices.SortFunc(keys, p.compareMapKeys)

			if p.PrintTypes != PrintTypesNever {
				p.printString(p.valueTypeString(v))
			}

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

	if p.PrintTypes != PrintTypesNever {
		p.printString(vt.String())
	}

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
	if p.PrintTypes != PrintTypesNever {
		p.printByte('(')
		p.printString(p.valueTypeString(v))
		p.printByte(')')

		p.printByte('(')
	}

	p.printPointerAddressValue(v.Pointer())

	if p.PrintTypes != PrintTypesNever {
		p.printByte(')')
	}
}

func (p *Printer) printFunctionValue(v reflect.Value) {
	if p.PrintTypes != PrintTypesNever {
		p.printByte('(')
		p.printString(p.valueTypeString(v))
		p.printByte(')')

		p.printByte('(')
	}

	p.printPointerAddressValue(v.Pointer())

	if p.PrintTypes != PrintTypesNever {
		p.printByte(')')
	}
}

func (p *Printer) printInterfaceValue(v reflect.Value) {
	if v.IsZero() {
		if p.PrintTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.PrintTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		p.printValue(v.Elem())
	}
}

func (p *Printer) printPointerValue(v reflect.Value) {
	if v.IsZero() {
		if p.PrintTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.PrintTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		if p.checkPointer(v.Pointer()) {
			if p.PrintTypes != PrintTypesNever {
				p.printByte('&')
			}

			p.printValue(v.Elem())
		}
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

func (p *Printer) valueTypeString(v reflect.Value) string {
	s := v.Type().String()

	// It does not seem possible to get the actual interface type behind a
	// variable. I.e. reflect.TypeOf(any(42)).Kind is reflect.Int, not
	// reflect.interface. So we do something really ugly. But it works. Blame
	// Go.
	s = strings.ReplaceAll(s, "interface {}", "any")

	return s
}
