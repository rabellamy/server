# gRPC Example

This example demonstrates how to use the `grpc` package to create a gRPC server with built-in metrics, health checks, and reflection, along with a simple "Hello World" service.

## Components

- **Server**: Located in `main.go`, it sets up the `Greeter` service.
- **Client**: Located in `client/main.go`, it connects to the server and calls the `SayHello` method.
- **Proto**: The service definition is in `helloworld/helloworld.proto`.

## Running the Example

### 1. Start the Server

From the root of the repository, run:

```bash
go run examples/grpc/main.go
```

The server will start with the following defaults (unless overridden by environment variables):
- **gRPC API**: `0.0.0.0:50051`
- **Metrics HTTP**: `0.0.0.0:2112/metrics`

To override defaults, use the `TEST_` prefix:
```bash
TEST_APIHOST=0.0.0.0:9090 go run examples/grpc/main.go
```

### 2. Run the Client

In a separate terminal, run:

```bash
go run examples/grpc/client/main.go
```

You can specify the address and a name to greet using flags:

```bash
go run examples/grpc/client/main.go -addr localhost:50051 -name Sally
```

## Features Demonstrated

1. **Automatic Health Checks**: The server automatically registers the gRPC Health Checking Service.
2. **Metrics**: gRPC metrics are automatically collected and exposed via the metrics HTTP server.
3. **Reflection**: gRPC Reflection is enabled for debugging with tools like `grpcurl`.
4. **Graceful Shutdown**: The server handles `SIGINT` and `SIGTERM` for graceful stops.
