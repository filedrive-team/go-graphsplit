build:
	rm -rf ./graphsplit
	go build -ldflags "-s -w" -o graphsplit ./cmd/graphsplit/main.go
.PHONY: build

## FFI

ffi: 
	./extern/filecoin-ffi/install-filcrypto
.PHONY: ffi

test:
	go test -v ./...
.PHONY: test
