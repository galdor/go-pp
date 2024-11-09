package main

import (
	"fmt"
	"reflect"

	"go.n16f.net/pp"
)

type Id struct {
	Type  string
	Value int
}

func (id Id) String() string {
	return fmt.Sprintf("%s:%d", id.Type, id.Value)
}

func FormatValue(v reflect.Value) any {
	if id, ok := v.Interface().(Id); ok {
		return pp.RawString(id.String())
	}

	return pp.FormatValue(v)
}

func main() {
	pp.Print(Id{"user", 42}, "default format")

	pp.DefaultPrinter.SetFormatValueFunc(FormatValue)
	pp.Print(Id{"user", 42}, "custom format")
}
