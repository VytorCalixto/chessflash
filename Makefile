APP_NAME=chessflash
GOFILES=$(shell find . -name "*.go" -not -path "./vendor/*")

.PHONY: run
run:
	go run ./cmd/server

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test ./...

.PHONY: fmt
fmt:
	go fmt ./...


