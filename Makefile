
build:
	go build -o bin/redis .


run: build
	./bin/redis


test:
	go test -v ./...