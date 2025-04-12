# Paprika 3 MCP Server

Paprika is a tool designed to help manage recipes, plan meals, and organize grocery lists. The Paprika 3 MCP Server provides a connection to the Paprika sync API, allowing interaction with your recipe collection.

> **Note:** Some features may not yet be available.

## Prerequisites

- Go 1.x or higher
- `jq` command-line tool (for debugging)

## Building

You can build the project using Make:

```bash
make build
```

Or manually with Go:

```bash
go build ./cmd/paprika-3-mcp
```

## Running as Part of an MCP Client

To use this server as part of an MCP client, such as Claude Desktop, you need to compile it first (refer to the [Building](#building) section). After compilation, configure the client to use your Paprika username and password. Below is an example configuration for Claude Desktop:

```json
{
    "mcpServers": {
        "paprika": {
            "command": "/Users/yourusername/path-to-download-directory/paprika-3-mcp/paprika-3-mcp",
            "env": {
                "PAPRIKA_USERNAME": "your_email",
                "PAPRIKA_PASSWORD": "your_password"
            }
        }
    }
}
```

### Steps to Configure:

1. Replace `/Users/yourusername/path-to-download-directory/` with the actual path where the `paprika-3-mcp` binary is located.
2. Update `your_email` and `your_password` with your Paprika account credentials.
3. Save the configuration file in the appropriate location for your MCP client.

Once configured, the MCP client will use the Paprika 3 MCP Server to interact with your recipe collection.

## Development

The project includes several Make targets to help with development:

- `make test` - Run all tests
- `make clean` - Clean build artifacts
- `make debug-tools` - List available JSON-RPC tools
- `make debug-recipes` - List recipe summaries

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
    }' | jq -c | go run . | jq .
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
    }' | jq -c | go run . \
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
    }' | jq -c | go run . \
       | jq '.result.contents[0].text | fromjson'
    ```
