.PHONY: build test lint clean run

build:
	go build -o bin/goperf .

test:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/ coverage.out

run: build
	./bin/goperf run -n 3 https://httpbin.org/get
