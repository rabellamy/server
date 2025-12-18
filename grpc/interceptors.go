package grpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rabellamy/promstrap/strategy"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryREDInterceptor returns a gRPC unary interceptor that records RED metrics.
func UnaryREDInterceptor(red *strategy.RED) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		service, method, err := extractServiceMethod(info.FullMethod)
		if err != nil {
			return nil, err
		}

		// Record the request (Rate)
		red.Requests.WithLabelValues(service, method).Inc()

		// Call the handler
		resp, err := handler(ctx, req)

		// Record duration
		duration := time.Since(start).Seconds()
		if red.Duration.Histogram != nil {
			red.Duration.Histogram.WithLabelValues(service, method).Observe(duration)
		}
		if red.Duration.Summary != nil {
			red.Duration.Summary.WithLabelValues(service, method).Observe(duration)
		}

		// Record errors
		if err != nil {
			st, _ := status.FromError(err)
			red.Errors.WithLabelValues(st.Code().String()).Inc()
		}

		return resp, err
	}
}

// StreamREDInterceptor returns a gRPC stream interceptor that records RED metrics.
// Note: This only records the start of the stream as a request and the final status as an error if applicable.
// True stream metrics often require more granular tracking (messages sent/received).
func StreamREDInterceptor(red *strategy.RED) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		service, method, err := extractServiceMethod(info.FullMethod)
		if err != nil {
			return err
		}

		// Record the request (Rate)
		red.Requests.WithLabelValues(service, method).Inc()

		// Call the handler
		err = handler(srv, ss)
		if err != nil {
			st, _ := status.FromError(err)
			red.Errors.WithLabelValues(st.Code().String()).Inc()
		}

		// Record duration
		duration := time.Since(start).Seconds()
		if red.Duration.Histogram != nil {
			red.Duration.Histogram.WithLabelValues(service, method).Observe(duration)
		}
		if red.Duration.Summary != nil {
			red.Duration.Summary.WithLabelValues(service, method).Observe(duration)
		}

		// Record errors
		if err != nil {
			st, _ := status.FromError(err)
			red.Errors.WithLabelValues(st.Code().String()).Inc()
		}

		return err
	}
}

// Extract service and method from FullMethod (e.g., "/helloworld.Greeter/SayHello")
func extractServiceMethod(fullMethod string) (string, string, error) {
	if !strings.HasPrefix(fullMethod, "/") {
		return "", "", fmt.Errorf("invalid gRPC method format: %s", fullMethod)
	}

	lastSlash := strings.LastIndex(fullMethod, "/")
	if lastSlash <= 0 {
		return "", "", fmt.Errorf("invalid gRPC method format: %s", fullMethod)
	}

	service := fullMethod[1:lastSlash]
	method := fullMethod[lastSlash+1:]

	if service == "" {
		return "", "", fmt.Errorf("invalid gRPC method: service name missing in %s", fullMethod)
	}

	if method == "" {
		return "", "", fmt.Errorf("invalid gRPC method: method name missing in %s", fullMethod)
	}

	return service, method, nil
}
