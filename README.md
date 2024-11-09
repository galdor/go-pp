# go-pp
## Introduction
The go-pp library, "pp" being for "pretty printer" contains utilities to pretty
print Go values. Its main use case is to help debugging. The `%#v` standard
escape sequence is fundamentally user hostile: no indentation, prints pointers
instead of recursing, expands values that should not be expanded (e.g.
`time.Time` values).

The `pp` package aims to make your life easier.

## Usage
Refer to the [Go package documentation](https://pkg.go.dev/go.n16f.net/pp)
for information about the API.

# Licensing
Go-pp is open source software distributed under the
[ISC](https://opensource.org/licenses/ISC) license.

