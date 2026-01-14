main:
	go run main.go		

test:
	 go test ./... 
lint:
	golangci-lint run
