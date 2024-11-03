package pp

import (
	"io"
)

var DefaultPrinter Printer

func Print(value any) error {
	return DefaultPrinter.Print(value)
}

func PrintTo(value any, w io.Writer) error {
	return DefaultPrinter.PrintTo(value, w)
}

func String(value any) string {
	return DefaultPrinter.String(value)
}
