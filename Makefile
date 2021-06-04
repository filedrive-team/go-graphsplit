.PHONY: build
build:
	go build -ldflags "-s -w" -o graphsplit ./cmd/graphsplit/main.go


## FFI

ffi: 
	./extern/filecoin-ffi/install-filcrypto
.PHONY: ffi



