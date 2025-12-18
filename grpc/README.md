# grpc

`grpc` is a Go package that provides a production-ready gRPC server with built-in observability, configuration management, and graceful shutdown capabilities.

## Features

- **Graceful Shutdown**: Handles OS signals (SIGINT, SIGTERM) to shut down the server gracefully, waiting for active RPCs to complete (during shutdown timeout, then force stops).
- **Observability**:
    - **Prometheus Metrics**: Exposes a dedicated `/metrics` endpoint on a separate port/goroutine (default 2112).
    - **Interceptors**: Includes standard interceptors for metrics (unary/stream).
- **Health Check**: Implements standard gRPC health check service.
- **Configuration**: Easy configuration via environment variables using  [`envconfig`](https://github.com/kelseyhightower/envconfig).
- **Structured Logging**: Uses `log/slog` for structured logging.

## Usage

Here's a basic example of how to use `grpc`:

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/rabellamy/server/grpc"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	// 1. Create Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 2. Load Configuration
	config, err := grpc.LoadConfig("test")
	if err != nil {
		logger.Error("config loading failed", "err", err)
		os.Exit(1)
	}

	// 3. Define Registration Function
	register := func(s *googlegrpc.Server) {
        // Register your services here
		// pb.RegisterGreeterServer(s, &server{})
	}

	// 4. Create Server
	server, err := grpc.NewServer(context.Background(), config, register, logger)
	if err != nil {
		logger.Error("server instantiation failed", "err", err)
		os.Exit(1)
	}

	// 5. Run Server
	if err := server.Run(); err != nil {
		logger.Error("server startup failed", "err", err)
		os.Exit(1)
	}
}
```

## Testing

The server has **gRPC Reflection** enabled, allowing you to use tools like [`grpcurl`](https://github.com/fullstorydev/grpcurl) to interact with it.

### Listing Services
```bash
grpcurl -plaintext localhost:50051 list
```

### Health Checks
The server implements the standard gRPC Health Checking Protocol.

**Check overall health:**
```bash
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check
```

**Check a specific service (default name is "test"):**
```bash
grpcurl -plaintext -d '{"service": "test"}' localhost:50051 grpc.health.v1.Health/Check
```


## Configuration

The server is configured using environment variables.

| Field | Environment Variable | Default | Description |
|-------|--------------------------------------|---------|-------------|
| `ShutdownTimeout` | `APP_SHUTDOWNTIMEOUT` | `20s` | Maximum duration to wait for graceful shutdown before forcing stop. |
| `APIHost` | `APP_APIHOST` | `0.0.0.0:50051` | Host and port for the gRPC server. |
| `DebugHost` | `APP_DEBUGHOST` | `0.0.0.0:3010` | Host and port for debug endpoints (if used). |
| `MetricsHost` | `APP_METRICSHOST` | `0.0.0.0:2112` | Host and port for the Prometheus metrics server. |
| `Build` | `APP_BUILD` | `dev` | Build version/tag. |
| `Desc` | `APP_DESC` | `example grpc server` | Server description. |
| `Namespace` | `APP_NAMESPACE` | `APP` | Namespace for metrics. |
