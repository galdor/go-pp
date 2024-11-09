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

type Point struct {
	X int
	Y int
	Z int
}

type Complex struct {
	Points []Point
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

	// Inline content
	pp.Print(Complex{
		Points: []Point{
			{1, -20, +300},
			{26602, 31921, 19128},
			{23902, 3278, 2333527093},
		},
	})
}
