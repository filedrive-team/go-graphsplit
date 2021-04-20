.PHONY: build
build:
	go build -ldflags "-s -w" -o graphsplit graphsplit.go utils.go retrieve.go
