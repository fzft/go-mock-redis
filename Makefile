
build:
	go build -o bin/redis .


run: build
	./bin/redis