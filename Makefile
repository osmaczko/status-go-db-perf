.PHONY: build run

build:
	mkdir -p ./build
	go build -o ./build/status-go-db-perf .

run: build
	./build/status-go-db-perf
