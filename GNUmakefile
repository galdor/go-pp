all: build

build:
	go build -o bin/ $(CURDIR)/...

check: vet

vet:
	go vet $(CURDIR)/...

test:
	go test $(CURDIR)/...

.PHONY: all build check vet test
