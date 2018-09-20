.PHONY: format
format:
	gofmt -d -w -s -e ./bin/ ./config/

.PHONY: test
test:
	go test ./...
