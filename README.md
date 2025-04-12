# Paprika 3 MCP Server

Paprika is a tool designed to help manage recipes, plan meals, and organize grocery lists. The Paprika 3 MCP Server provides a connection to the Paprika sync API, allowing interaction with your recipe collection.

> **Note:** Some features may not yet be available.

## Prerequisites

- Go 1.x or higher
- `jq` command-line tool (for debugging)

## Building

```bash
go build ./cmd/paprika-3-mcp
```

## Running

```bash
export PAPRIKA_USERNAME="your_email"
export PAPRIKA_PASSWORD="your_password"
./paprika-3-mcp
```

The server will start on port 8080 by default.

## Debugging

Follow these steps to send JSON-RPC requests to the server using the CLI:

1. Set your credentials:
    ```bash
    export PAPRIKA_USERNAME="your_email"
    export PAPRIKA_PASSWORD="your_password"
    ```

2. Navigate to the command directory:
    ```bash
    cd cmd/paprika-3-mcp/
    ```

3. Send a request to list tools:
    ```bash
    echo '{
        "jsonrpc": "2.0",
        "method": "tools/list",
        "params": {},
        "id": 1
    }' | jq -c \
       | go run . \
       | jq .
    ```

4. Send a request to call a tool (e.g., `list_recipe_summaries`):
    ```bash
    echo '{
        "jsonrpc": "2.0",
        "method": "tools/call",
        "params": {
          "name": "list_recipe_summaries"
        },
        "id": 1
    }' | jq -c \
       | go run . \
       | jq '.result.content[1].resource.text | fromjson'
    ```

5. Send a request to get a resource
    ```bash
    echo '{
        "jsonrpc": "2.0",
        "method": "resources/read",
        "params": {
          "uri": "paprika://recipes/A2FDA12F-AB23-1234-AB11-465E530B0B42"
        },
        "id": 1
    }' | jq -c \
       | go run . \
       | jq '.result.contents[0].text | fromjson'
    ```
