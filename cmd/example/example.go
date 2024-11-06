package main

import (
	"time"

	"github.com/galdor/go-pp"
)

type Foo struct {
	Foo  *Foo
	Foos []*Foo
	Bar  *Bar
}

type Bar struct {
	Foo *Foo
}

func main() {
	pp.Print(time.Now(), "now")

	// Cyclic pointers
	foo1 := Foo{}
	foo2 := Foo{Foo: &foo1}
	bar := Bar{Foo: &foo2}

	foo1.Foo = &foo1
	foo1.Foos = []*Foo{&foo1, &foo2}
	foo1.Bar = &bar

	pp.Print(&foo1)

	// Cyclic slices
	var ns []any

	ns = append(ns, 1)
	ns = append(ns, 2)
	ns = append(ns, 3)
	ns[0] = ns
	ns[2] = ns

	pp.Print(ns)
}
