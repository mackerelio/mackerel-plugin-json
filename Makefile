setup:
	go install github.com/Songmu/goxz/cmd/goxz@latest
	go install github.com/tcnksm/ghr@latest
	go install golang.org/x/lint/golint@latest
	go get -d -t ./...

test: setup
	go test -v ./...

lint: setup
	go vet ./...
	golint -set_exit_status ./...

.PHONY: setup test lint
