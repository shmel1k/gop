all:
	GO111MODULE=on go mod vendor
	GO111MODULE=on go mod tidy
	go test ./... -race -cover
