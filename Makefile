.PHONY: build test fmt vet

build:
	CGO_ENABLED=0 go build -o iptables-comment-exporter .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l .
