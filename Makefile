.PHONY: build install test clean run debug-tools debug-recipes

build: 
	go build -o bin ./cmd/...

install:
	go install ./cmd/paprika-3-mcp

# Run all tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run the application
run: build
	./bin/paprika-3-mcp

# Debug target that lists available tools
debug-tools:
	echo '{"jsonrpc": "2.0", "method": "tools/list", "params": {}, "id": 1}' | \
	jq -c | \
	go run ./cmd/paprika-3-mcp | \
	jq .

# Debug target that lists recipe summaries
debug-recipes:
	echo '{"jsonrpc": "2.0", "method": "tools/call", "params": {"name": "list_recipe_summaries"}, "id": 1}' | \
	jq -c | \
	go run ./cmd/paprika-3-mcp | \
	jq '.result.content[1].resource.text | fromjson'
