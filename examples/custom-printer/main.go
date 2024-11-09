package main

import (
	"os"

	"go.n16f.net/pp"
)

func main() {
	var p pp.Printer

	p.SetDefaultOutput(os.Stderr)
	p.SetIndent("\t")
	p.SetLinePrefix("> ")

	info, _ := os.Stat("/dev/stdout")
	p.Print(info, "stdout info")
}
