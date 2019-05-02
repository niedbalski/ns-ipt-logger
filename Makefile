build:
	go mod vendor
	go mod download
	go build -o ipt-ns-logger-$(git rev-parse --short HEAD)

