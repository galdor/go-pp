package main

import (
	"net/http"

	"github.com/galdor/go-pp"
)

func main() {
	pp.Print(http.DefaultServeMux)
}
