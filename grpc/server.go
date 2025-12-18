package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rabellamy/server/metrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	grpcServer    *grpc.Server
	healthServer  *health.Server
	metricsServer http.Server
	ctx           context.Context
	logger        *slog.Logger
	config        Config
}

type RegisterFunc func(*grpc.Server)

func NewServer(ctx context.Context, config Config, register RegisterFunc, logger *slog.Logger, opts ...grpc.ServerOption) (*Server, error) {
	// Enable gRPC metrics
	grpcMetrics := grpc_prometheus.NewServerMetrics()

	// Custom RED interceptors using promstrap
	red, err := metrics.NewRED(config.Namespace, "grpc", []string{"service", "method"}, []string{"service", "method"})
	if err != nil {
		return nil, fmt.Errorf("failed to create RED metrics: %w", err)
	}
	if err := red.Register(); err != nil {
		return nil, fmt.Errorf("failed to register RED metrics: %w", err)
	}

	// Default interceptors
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			grpcMetrics.UnaryServerInterceptor(),
			UnaryREDInterceptor(red),
		),
		grpc.ChainStreamInterceptor(
			grpcMetrics.StreamServerInterceptor(),
			StreamREDInterceptor(red),
		),
	)

	s := grpc.NewServer(opts...)

	// Register services
	if register != nil {
		register(s)
	}

	// Register reflection for debugging
	reflection.Register(s)

	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)

	// Initialize metrics
	grpcMetrics.InitializeMetrics(s)

	// Metrics HTTP server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	server := &Server{
		grpcServer:   s,
		healthServer: healthServer,
		metricsServer: http.Server{
			Addr:    config.MetricsHost,
			Handler: metricsMux,
		},
		logger: logger,
		ctx:    ctx,
		config: config,
	}

	return server, nil
}

func (s *Server) Run() error {
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	return s.run(shutdown)
}

func (s *Server) run(shutdown <-chan os.Signal) error {
	serverErrors := make(chan error, 2)

	// Start metrics server
	go func() {
		s.logger.Info("startup", "status", "metrics server started", "host", s.config.MetricsHost)
		serverErrors <- s.metricsServer.ListenAndServe()
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", s.config.APIHost)
		if err != nil {
			serverErrors <- fmt.Errorf("failed to listen on %s: %w", s.config.APIHost, err)
			return
		}
		s.logger.Info("startup", "status", "grpc server started", "host", s.config.APIHost)

		// Set serving status to SERVING
		s.healthServer.SetServingStatus(s.config.Name, grpc_health_v1.HealthCheckResponse_SERVING)

		serverErrors <- s.grpcServer.Serve(lis)
	}()

	select {
	case <-s.ctx.Done():
		// Create a new context for shutdown to allow for graceful stop even if the parent context is cancelled
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
		defer cancel()
		return s.shutdownServers(shutdownCtx, nil)
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		ctx, cancel := context.WithTimeout(s.ctx, s.config.ShutdownTimeout)
		defer cancel()
		return s.shutdownServers(ctx, sig)
	}
}

func (s *Server) shutdownServers(ctx context.Context, signal os.Signal) error {
	// We can assume that if the signal is nil, it is context cancelled
	// by internal application logic
	sig := "context_cancelled"
	if signal != nil {
		sig = signal.String()
	}

	s.logger.Info("shutdown", "server", "health", "status", "shutdown complete", "signal", sig)

	// Set serving status to NOT_SERVING
	s.healthServer.SetServingStatus(s.config.Name, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// GracefulStop for gRPC doesn't take a context, it waits indefinitely or until connections drain.
	// To respect the shutdown timeout, we can wrap it in a goroutine/channel.
	s.logger.Info("shutdown", "server", "grpc", "status", "shutting down started", "signal", sig)
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Shutdown metrics server
	s.logger.Info("shutdown", "server", "metrics", "status", "shutdown started", "signal", sig)
	if err := s.metricsServer.Shutdown(ctx); err != nil {
		s.metricsServer.Close()
		return fmt.Errorf("metrics server could not stop gracefully: %w", err)
	}
	s.logger.Info("shutdown", "server", "metrics", "status", "shutdown complete", "signal", sig)

	select {
	case <-ctx.Done():
		// Force stop if timeout exceeded
		s.grpcServer.Stop()
		return fmt.Errorf("grpc server shutdown timed out")
	case <-stopped:
		s.logger.Info("shutdown", "server", "grpc", "status", "graceful stop complete", "signal", sig)
	}

	return nil
}
