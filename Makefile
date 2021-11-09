.PHONY: build
build:
	@GOOS=linux GOARCH=amd64 go build -ldflags='-X main.version=1.0.0's -o bin/turnipmon .