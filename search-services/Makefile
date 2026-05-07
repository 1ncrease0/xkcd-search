lint: protolint golint

protobuf:
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/words/words.proto
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/update/update.proto
	protoc --go_out=. --go_opt=paths=source_relative \
               --go-grpc_out=. --go-grpc_opt=paths=source_relative \
               proto/search/search.proto

protolint:
	protolint .

golint:
	golangci-lint run -E gocritic -v ./...

mocks:
	go run github.com/vektra/mockery/v2@latest --config .mockery.yaml

test:
	go test -race -coverprofile cover.out \
		$(shell go list ./... | grep -E -v '(/mocks($$|/)|/proto/|github.com/1ncrease0/xkcd-search/(api|update|words|search)$$)')
	go tool cover -func cover.out | tail -1
	go tool cover -html cover.out -o cover.html