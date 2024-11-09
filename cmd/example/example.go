package main

import (
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

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

var nbTitles = 0

func printTitle(s string) {
	if nbTitles > 0 {
		fmt.Println()
	}

	fmt.Println(s)
	for range utf8.RuneCountInString(s) {
		fmt.Print("-")
	}
	fmt.Println()

	nbTitles++
}

func main() {
	// Standard types
	printTitle("STANDARD TYPES")

	pp.Print(map[string]any{
		"timestamp": time.Now(),
		"regexp":    regexp.MustCompile("^(?i)hell(o+)$"),
	})

	// Pointer handling
	printTitle("REFERENCES")

	foo1 := Foo{}
	foo2 := Foo{Foo: &foo1}
	bar := Bar{Foo: &foo2}

	foo1.Foo = &foo1
	foo1.Foos = []*Foo{&foo1, &foo2}
	foo1.Bar = &bar

	pp.Print(&foo1)

	// Inline content
	printTitle("INLINE CONTENT")

	pp.Print(Complex{
		Points: []Point{
			{1, -20, +300},
			{26602, 31921, 19128},
			{23902, 3278, 2333527093},
		},
	})
}
