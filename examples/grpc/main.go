package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/rabellamy/server/config"
	"github.com/rabellamy/server/examples/grpc/helloworld"
	"github.com/rabellamy/server/grpc"
	googlegrpc "google.golang.org/grpc"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	helloworld.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	return &helloworld.HelloReply{Message: fmt.Sprintf("Hello %s", in.GetName())}, nil
}

func main() {
	// 1. Create Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// 2. Load Configuration (prefix 'test' means env vars like TEST_APIHOST)
	config, err := config.LoadConfig[grpc.Config]("test")
	if err != nil {
		logger.Error("config loading failed", "err", err)
		os.Exit(1)
	}

	// 3. Define Registration Function
	register := func(s *googlegrpc.Server) {
		helloworld.RegisterGreeterServer(s, &server{})
		logger.Info("registering services", "service", "Greeter")
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
