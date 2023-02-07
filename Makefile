setup:
	go install golang.org/x/lint/golint@latest
	go mod download

test: setup
	go test -v ./...

lint: setup
	go vet ./...
	golint -set_exit_status ./...

.PHONY: setup test lint
