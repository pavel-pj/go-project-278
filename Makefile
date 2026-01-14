main:
	go run main.go		

test:
	 go test -race ./...
lint:
	golangci-lint run --timeout=5m ./...

## build: Собрать бинарник
build:
	 go build -o app -v ./...

