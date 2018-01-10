setup:
	go get \
		github.com/Songmu/goxz/cmd/goxz \
		github.com/tcnksm/ghr \
		github.com/golang/lint/golint
	go get -d -t ./...

test: setup
	go test -v ./...

lint: setup
	go vet ./...
	golint -set_exit_status ./...

.PHONY: setup test lint
