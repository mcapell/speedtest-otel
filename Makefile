.PHONY: verify vet test

verify:
	go mod verify

vet:
	go vet ./...

test: verify vet
	go test -v ./...
