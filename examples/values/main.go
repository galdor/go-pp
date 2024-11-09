package main

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"go.n16f.net/pp"
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
	pp.Print(os.Args)

	pp.Print(os.Args, "command line arguments")
	return

	pp.DefaultPrinter.SetLinePrefix("> ")

	// Standard types
	printTitle("STANDARD TYPES")

	pp.Print(42, "integer")
	pp.Print(math.E, "float")
	pp.Print("Hello world!\n", "string")
	pp.Print(time.Now(), "timestamp")
	pp.Print(2*time.Hour+15*time.Minute+42250*time.Millisecond, "duration")
	pp.Print(regexp.MustCompile("^(?i)hell(o+)$"), "regexp")
	pp.Print(big.NewRat(248311, 179), "rational")

	var av atomic.Value
	av.Store(42)
	pp.Print(av, "atomic value")

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
	}, "complex value")
}
