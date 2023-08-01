# go-mock-redis

`go-mock-redis` is a simplified, single-threaded implementation of Redis in Go. It is designed to serve as a learning tool and a mock Redis server for testing purposes.

## Features

- **Single-threaded**: The entire server runs in a single goroutine, emulating Redis's single-threaded nature. This ensures that commands are processed sequentially, providing a simple and predictable execution model.
- **Pipelines and Multiplexers**: `go-mock-redis` supports pipelining, allowing clients to send multiple commands to the server without waiting for each response. This can significantly improve performance by reducing network latency.
- **Data Structures**: `go-mock-redis` implements various Redis data structures, including strings, queues, sets, and sorted sets (zsets).