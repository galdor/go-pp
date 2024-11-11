package main

import (
	"os"

	"go.n16f.net/pp"
)

func main() {
	info, _ := os.Stat("/dev/stdout")
	pp.Print(info, "standard output")
}
