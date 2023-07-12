build:
	@go build -o bin/app cmd/main.go


build-win:
	@go build -o bin/app.exe cmd/main.go

run: build
	@./bin/app
	rm -rf ./bun/app

run-win: build
	@./bin/app.exe

test:
	@go test ./...