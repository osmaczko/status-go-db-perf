.PHONY: build run

build:
	mkdir -p ./build
	go build -o ./build/status-go-db-perf .

run: build
	./build/status-go-db-perf -dbPath=$(DB_PERF_PATH) -key=$(DB_PERF_KEY)
