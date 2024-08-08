build:
	rm -rf ./graphsplit
	go build -ldflags "-s -w" -o graphsplit ./cmd/graphsplit/main.go
.PHONY: build

test:
	go test -v ./...
.PHONY: test
