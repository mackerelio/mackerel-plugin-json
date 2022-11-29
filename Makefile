setup:
	go install \
		github.com/Songmu/goxz/cmd/goxz@latest \
		github.com/tcnksm/ghr@latest \
		golang.org/x/lint/golint@latest
	go get -d -t ./...

test: setup
	go test -v ./...

lint: setup
	go vet ./...
	golint -set_exit_status ./...

.PHONY: setup test lint
