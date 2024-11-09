package main

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sync/atomic"
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

	var bi big.Int
	bi.SetString("-8789765579753643555083504787829125689141207431643136", 10)

	var av atomic.Value
	av.Store(42)

	pp.Print(map[string]any{
		"integer":   42,
		"float":     math.E,
		"string":    "Hello world!\n",
		"timestamp": time.Now(),
		"duration":  3*time.Hour + 15*time.Minute + 42*time.Second,
		"regexp":    regexp.MustCompile("^(?i)hell(o+)$"),
		"bignums": []any{
			bi,
			big.NewFloat(math.Pi),
			big.NewRat(248311, 179),
		},
		"atomic-value": av,
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
