build:
	@go build -o bin/app cmd/main.go

run: build
	@./bin/app
	rm -rf ./bun/app

test:
	@go test ./...