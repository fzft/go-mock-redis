# go-mock-redis

`go-mock-redis` is a simplified, single-threaded implementation of Redis in Go. It is designed to serve as a learning tool and a mock Redis server for testing purposes.

## Requirements
 - Go 1.20 or higher

## Features

- **Single-threaded**: The entire server runs in a single goroutine, emulating Redis's single-threaded nature. This ensures that commands are processed sequentially, providing a simple and predictable execution model.
- **Pipelines and Multiplexers**: `go-mock-redis` supports pipelining, allowing clients to send multiple commands to the server without waiting for each response. This can significantly improve performance by reducing network latency.
- **Data Structures**: `go-mock-redis` implements various Redis data structures, including strings, queues, sets, and sorted sets (zsets).
- **REPL**: `go-mock-redis` includes a REPL (read-eval-print loop) that allows users to interact with the server via a command line interface.
- **ZeroCopy**: `go-mock-redis` uses zero-copy techniques `sendfile` to avoid unnecessary memory allocations and copies. This improves performance and reduces memory usage.
- **RESP**: `go-mock-redis` uses the RESP3 (REdis Serialization Protocol) to communicate with clients. This allows it to be compatible with existing Redis clients.
## Building
