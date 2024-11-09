package pp

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"
)

type RawString string

type FormatValueFunc func(reflect.Value) any

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
	DefaultMaxInlineColumn    = 80
	DefaultIndent             = "  "
	DefaultThousandsSeparator = '_'
	DefaultFormatValueFunc    = FormatValue
)

type Printer struct {
	formatValue        FormatValueFunc
	maxInlineColumn    int
	indent             string
	linePrefix         string
	printTypes         PrintTypes
	hidePrivateFields  bool
	thousandsSeparator rune

	buf    []byte
	level  int
	inline bool

	pointers map[uintptr]*pointerRef

	mu sync.Mutex
}

type pointerRef struct {
	n       int
	printed bool
}

func (p *Printer) SetFormatValueFunc(fn FormatValueFunc) {
	p.mu.Lock()
	p.formatValue = fn
	p.mu.Unlock()
}

func (p *Printer) SetMaxInlineColumn(column int) {
	p.mu.Lock()
	p.maxInlineColumn = column
	p.mu.Unlock()
}

func (p *Printer) SetIndent(indent string) {
	p.mu.Lock()
	p.indent = indent
	p.mu.Unlock()
}

func (p *Printer) SetLinePrefix(prefix string) {
	p.mu.Lock()
	p.linePrefix = prefix
	p.mu.Unlock()
}

func (p *Printer) SetPrintTypes(types PrintTypes) {
	p.mu.Lock()
	p.printTypes = types
	p.mu.Unlock()
}

func (p *Printer) SetHidePrivateFields(hide bool) {
	p.mu.Lock()
	p.hidePrivateFields = hide
	p.mu.Unlock()
}

func (p *Printer) SetThousandsSeparator(sep rune) {
	p.mu.Lock()
	p.thousandsSeparator = sep
	p.mu.Unlock()
}

func (p *Printer) Print(value any, label ...any) error {
	return p.PrintTo(os.Stdout, value, label...)
}

func (p *Printer) PrintTo(w io.Writer, value any, label ...any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.reset(value)
	p.maybePrintLabel(label...)
	p.printValueLine(value)

	_, err := w.Write(p.buf)
	return err
}

func (p *Printer) String(value any, label ...any) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.reset(value)
	p.maybePrintLabel(label...)
	p.printValueLine(value)

	return string(p.buf)
}

func (p *Printer) clone() *Printer {
	p2 := Printer{
		formatValue:        p.formatValue,
		maxInlineColumn:    p.maxInlineColumn,
		indent:             p.indent,
		linePrefix:         p.linePrefix,
		printTypes:         p.printTypes,
		hidePrivateFields:  p.hidePrivateFields,
		thousandsSeparator: p.thousandsSeparator,

		level:  p.level,
		inline: p.inline,

		pointers: p.pointers,
	}

	return &p2
}

func (p *Printer) reset(value any) {
	if p.formatValue == nil {
		p.formatValue = FormatValue
	}

	if p.maxInlineColumn == 0 {
		p.maxInlineColumn = DefaultMaxInlineColumn
	}

	if p.indent == "" {
		p.indent = DefaultIndent
	}

	if p.printTypes == "" {
		p.printTypes = PrintTypesDefault
	}

	if p.thousandsSeparator == 0 {
		p.thousandsSeparator = DefaultThousandsSeparator
	}

	p.buf = nil

	p.initPointers(reflect.ValueOf(value))
}

func (p *Printer) initPointers(v reflect.Value) {
	p.pointers = make(map[uintptr]*pointerRef)

	visitedPointers := make(map[uintptr]struct{})

	var fn func(reflect.Value)
	fn = func(v reflect.Value) {
		vt := v.Type()

		switch v.Kind() {
		case reflect.Map, reflect.Slice, reflect.Struct, reflect.Pointer:
		default:
			return
		}

		if v.Kind() != reflect.Struct {
			if v.IsNil() {
				return
			}

			ptr := v.Pointer()
			if _, found := visitedPointers[ptr]; found {
				p.pointers[ptr] = &pointerRef{n: len(p.pointers) + 1}
				return
			}
			visitedPointers[ptr] = struct{}{}
		}

		switch v.Kind() {
		case reflect.Map:
			iter := v.MapRange()
			for iter.Next() {
				fn(iter.Value())
			}

		case reflect.Slice:
			for i := range v.Len() {
				fn(v.Index(i))
			}

		case reflect.Struct:
			for i := range v.NumField() {
				fv := v.Field(i)
				ft := vt.Field(i)

				if !ft.IsExported() && p.hidePrivateFields {
					return
				}

				fn(fv)
			}

		case reflect.Pointer:
			if !v.IsZero() {
				fn(v.Elem())
			}
		}
	}

	fn(v)
}

func (p *Printer) pointerAnnotation(ptr uintptr) (bool, string) {
	ref, found := p.pointers[ptr]
	if !found {
		return false, ""
	}

	if !ref.printed {
		ref.printed = true
		return true, "#" + strconv.Itoa(ref.n) + "="
	}

	return false, "#" + strconv.Itoa(ref.n) + "#"
}

func (p *Printer) currentMaxInlineColumn() int {
	return p.maxInlineColumn - len(p.linePrefix) - p.level*len(p.indent)
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

	inlinable := p.inlinableValue(v)

	if inlinable && !p.inline {
		p2 := p.clone()

		p2.inline = true
		p2.printValue(v)
		data := p2.buf
		p.inline = false

		if utf8.RuneCount(data) <= p.currentMaxInlineColumn() {
			p.printBytes(data)
			return
		}
	}

	// Formatting function can return values which are themselves formattable.
	// So we iterate until we get to a value we cannot format.
	for {
		if !v.CanInterface() || p.formatValue == nil {
			break
		}

		var vs any
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
			if !v.IsNil() {
				vs = p.formatValue(v.Elem())
			}
		} else {
			vs = p.formatValue(v)
		}

		if vs == nil {
			break
		}

		if s, ok := vs.(RawString); ok {
			p.printValueString(v, string(s))
			return
		}

		v = reflect.ValueOf(vs)
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
	p.printString(p.linePrefix)

	for range p.level {
		p.printString(p.indent)
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
	if p.printTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	if b := v.Bool(); b {
		p.printString("true")
	} else {
		p.printString("false")
	}

	if p.printTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printIntegerValue(v reflect.Value) {
	if p.printTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	i := v.Int()
	s := strconv.FormatInt(i, 10)

	if p.thousandsSeparator == 0 {
		p.printString(s)
	} else {
		p.printString(p.addThousandsSeparator(s))
	}

	if p.printTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printUnsignedIntegerValue(v reflect.Value) {
	if p.printTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	u := v.Uint()
	s := strconv.FormatUint(u, 10)

	if p.thousandsSeparator == 0 {
		p.printString(s)
	} else {
		p.printString(p.addThousandsSeparator(s))
	}

	if p.printTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printFloatValue(v reflect.Value, bitSize int) {
	if p.printTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	f := v.Float()
	s := strconv.FormatFloat(f, 'f', -1, bitSize)

	is, fs, found := strings.Cut(s, ".")
	if found {
		if p.thousandsSeparator == 0 {
			p.printString(is)
		} else {
			p.printString(p.addThousandsSeparator(is))
		}

		p.printByte('.')

		p.printString(fs)
	} else {
		p.printString(s)
	}

	if p.printTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printComplexValue(v reflect.Value, bitSize int) {
	if p.printTypes == PrintTypesAlways {
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

	if p.printTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printStringValue(v reflect.Value) {
	if p.printTypes == PrintTypesAlways {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	s := v.String()
	buf := strconv.AppendQuote([]byte{}, s)
	p.printBytes(buf)

	if p.printTypes == PrintTypesAlways {
		p.printByte(')')
	}
}

func (p *Printer) printSequenceValue(v reflect.Value) {
	if v.Kind() == reflect.Slice && v.IsNil() {
		if p.printTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.printTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		if v.Kind() == reflect.Slice {
			first, annotation := p.pointerAnnotation(v.Pointer())
			if annotation != "" {
				p.printString(annotation)
				if !first {
					return
				}
			}
		}

		if p.printTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
		}

		p.printByte('[')
		if !p.inline {
			p.printNewline()
		}
		p.level++

		n := v.Len()
		for i := range n {
			ev := v.Index(i)

			if !p.inline {
				p.printLineStart()
			}

			p.printValue(ev)
			if !p.inline || i < n-1 {
				p.printByte(',')
			}

			if p.inline {
				if i < n-1 {
					p.printByte(' ')
				}
			} else {
				p.printNewline()
			}
		}

		p.level--
		if !p.inline {
			p.printLineStart()
		}
		p.printByte(']')
	}
}

func (p *Printer) printMapValue(v reflect.Value) {
	if v.IsNil() {
		if p.printTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.printTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		keys := v.MapKeys()

		if len(keys) == 0 {
			if p.printTypes != PrintTypesNever {
				p.printString(p.valueTypeString(v))
			}

			p.printString("{}")
			return
		}

		first, annotation := p.pointerAnnotation(v.Pointer())
		if annotation != "" {
			p.printString(annotation)
			if !first {
				return
			}
		}

		slices.SortFunc(keys, p.compareMapKeys)

		if p.printTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
		}

		p.printByte('{')
		if !p.inline {
			p.printNewline()
		}
		p.level++

		n := len(keys)
		i := 0
		for _, kv := range keys {
			vv := v.MapIndex(kv)

			if !p.inline {
				p.printLineStart()
			}

			p.printValue(kv)
			p.printString(": ")

			p.printValue(vv)
			if !p.inline || i < n-1 {
				p.printByte(',')
			}

			if p.inline {
				if i < n-1 {
					p.printByte(' ')
				}
			} else {
				p.printNewline()
			}

			i++
		}

		p.level--
		if !p.inline {
			p.printLineStart()
		}
		p.printByte('}')
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

	if p.printTypes != PrintTypesNever {
		p.printString(vt.String())
	}

	if vt.NumField() == 0 {
		p.printString("{}")
	} else {
		p.printByte('{')
		if !p.inline {
			p.printNewline()
		}
		p.level++

		n := vt.NumField()
		for i := range n {
			fv := v.Field(i)
			ft := vt.Field(i)

			if !ft.IsExported() && p.hidePrivateFields {
				continue
			}

			if !p.inline {
				p.printLineStart()
			}

			p.printString(ft.Name)
			p.printString(": ")

			p.printValue(fv)
			if !p.inline || i < n-1 {
				p.printByte(',')
			}

			if p.inline {
				if i < n-1 {
					p.printByte(' ')
				}
			} else {
				p.printNewline()
			}
		}

		p.level--
		if !p.inline {
			p.printLineStart()
		}
		p.printByte('}')
	}
}

func (p *Printer) printChannelValue(v reflect.Value) {
	if p.printTypes != PrintTypesNever {
		p.printByte('(')
		p.printString(p.valueTypeString(v))
		p.printByte(')')

		p.printByte('(')
	}

	p.printPointerAddressValue(v.Pointer())

	if p.printTypes != PrintTypesNever {
		p.printByte(')')
	}
}

func (p *Printer) printFunctionValue(v reflect.Value) {
	if p.printTypes != PrintTypesNever {
		p.printByte('(')
		p.printString(p.valueTypeString(v))
		p.printByte(')')

		p.printByte('(')
	}

	p.printPointerAddressValue(v.Pointer())

	if p.printTypes != PrintTypesNever {
		p.printByte(')')
	}
}

func (p *Printer) printInterfaceValue(v reflect.Value) {
	if v.IsZero() {
		if p.printTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.printTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		p.printValue(v.Elem())
	}
}

func (p *Printer) printPointerValue(v reflect.Value) {
	if v.IsZero() {
		if p.printTypes != PrintTypesNever {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printString("nil")

		if p.printTypes != PrintTypesNever {
			p.printByte(')')
		}
	} else {
		first, annotation := p.pointerAnnotation(v.Pointer())
		if annotation != "" {
			p.printString(annotation)
			if !first {
				return
			}
		}

		// The output for pointers to pointers can be confusing, so we print the
		// type by default.
		printType := p.printTypes == PrintTypesAlways ||
			(v.Elem().Kind() == reflect.Pointer &&
				p.printTypes != PrintTypesNever)

		if printType {
			p.printString(p.valueTypeString(v))
			p.printByte('(')
		}

		p.printByte('&')
		p.printValue(v.Elem())

		if printType {
			p.printByte(')')
		}
	}
}

func (p *Printer) printPointerAddressValue(ptr uintptr) {
	if ptr == 0 {
		p.printString("nil")
	} else {
		switch uintptrSize {
		case 4:
			p.printFormat("%#08x", ptr)
		case 8:
			p.printFormat("%#016x", ptr)
		default:
			p.printFormat("%#x", ptr)
		}
	}
}

func (p *Printer) printUnknownValue(v reflect.Value) {
	p.printFormat("%#v", v)
}

func (p *Printer) printValueString(v reflect.Value, s string) {
	if p.printTypes != PrintTypesNever {
		p.printString(p.valueTypeString(v))
		p.printByte('(')
	}

	p.printString(s)

	if p.printTypes != PrintTypesNever {
		p.printByte(')')
	}
}

func (p *Printer) valueTypeString(v reflect.Value) string {
	s := v.Type().String()

	// It does not seem possible to get the actual interface type behind a
	// variable. I.e. reflect.TypeOf(any(42)).Kind() is reflect.Int, not
	// reflect.interface. So we do something really ugly. But it works. Blame
	// Go.
	s = strings.ReplaceAll(s, "interface {}", "any")

	return s
}

func (p *Printer) addThousandsSeparator(s string) string {
	cs2 := make([]rune, len(s))

	cs := []rune(s)
	slices.Reverse(cs)

	for i, c := range cs {
		if i > 0 && i%3 == 0 {
			cs2 = append(cs2, p.thousandsSeparator)
		}

		cs2 = append(cs2, c)
	}

	slices.Reverse(cs2)

	return string(cs2)
}

func (p *Printer) inlinableValue(v reflect.Value) bool {
	if p.atomicValue(v) {
		return true
	}

	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := range v.Len() {
			if ev := v.Index(i); !p.atomicValue(ev) {
				return false
			}
		}

		return true

	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if !p.atomicValue(iter.Value()) {
				return false
			}
		}

		return true

	case reflect.Struct:
		for i := range v.NumField() {
			if fv := v.Field(i); !p.atomicValue(fv) {
				return false
			}
		}

		return true
	}

	return false
}

func (p *Printer) atomicValue(v reflect.Value) bool {
	atomicKinds := []reflect.Kind{
		reflect.Bool,
		reflect.Int,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr,
		reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128,
		reflect.String,
		reflect.Func, reflect.Chan,
		reflect.UnsafePointer,
	}

	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		return p.atomicValue(v.Elem())
	}

	return slices.Contains(atomicKinds, v.Kind())
}
