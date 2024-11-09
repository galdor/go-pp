package pp

import (
	"io"
)

var DefaultPrinter Printer

func Print(value any, label ...any) error {
	return DefaultPrinter.Print(value, label...)
}

func PrintTo(w io.Writer, value any, label ...any) error {
	return DefaultPrinter.PrintTo(w, value)
}
