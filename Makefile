build:
	go mod vendor
	go mod download
	go build -o ipt-ns-logger-$(shell git rev-parse --short HEAD)

