package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/rabellamy/server/metrics"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

func TestUnaryREDInterceptor(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace  string
		fullMethod string
		handler    grpc.UnaryHandler
		wantErr    bool
	}{
		"success": {
			namespace:  "test_unary_success",
			fullMethod: "/helloworld.Greeter/SayHello",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "ok", nil
			},
			wantErr: false,
		},
		"error": {
			namespace:  "test_unary_error",
			fullMethod: "/helloworld.Greeter/SayHello",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, errors.New("boom")
			},
			wantErr: true,
		},
		"invalid method": {
			namespace:  "test_unary_invalid",
			fullMethod: "/",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "ok", nil
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			red, err := metrics.NewRED(tt.namespace, "grpc", []string{"service", "method"}, []string{"service", "method"})
			assert.NoError(t, err)

			interceptor := UnaryREDInterceptor(red)
			info := &grpc.UnaryServerInfo{FullMethod: tt.fullMethod}
			_, err = interceptor(context.Background(), nil, info, tt.handler)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreamREDInterceptor(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace  string
		fullMethod string
		handler    grpc.StreamHandler
		wantErr    bool
	}{
		"success": {
			namespace:  "test_stream_success",
			fullMethod: "/helloworld.Greeter/SayHelloStream",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			wantErr: false,
		},
		"error": {
			namespace:  "test_stream_error",
			fullMethod: "/helloworld.Greeter/SayHelloStream",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return errors.New("boom")
			},
			wantErr: true,
		},
		"invalid method": {
			namespace:  "test_stream_invalid",
			fullMethod: "/",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			red, err := metrics.NewRED(tt.namespace, "grpc", []string{"service", "method"}, []string{"service", "method"})
			assert.NoError(t, err)

			interceptor := StreamREDInterceptor(red)
			info := &grpc.StreamServerInfo{FullMethod: tt.fullMethod}
			err = interceptor(nil, nil, info, tt.handler)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExtractServiceMethod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		fullMethod  string
		wantService string
		wantMethod  string
		wantErr     bool
		wantErrMsg  string
	}{
		"full format": {
			fullMethod:  "/helloworld.Greeter/SayHello",
			wantService: "helloworld.Greeter",
			wantMethod:  "SayHello",
			wantErr:     false,
		},
		"no leading slash": {
			fullMethod:  "helloworld.Greeter/SayHello",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
			wantErrMsg:  "invalid gRPC method format: helloworld.Greeter/SayHello",
		},
		"method only": {
			fullMethod:  "/SayHello",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
			wantErrMsg:  "invalid gRPC method format: /SayHello",
		},
		"invalid format: too many parts": {
			fullMethod:  "/too/many/parts",
			wantService: "too/many",
			wantMethod:  "parts",
			wantErr:     false,
		},
		"invalid format: just slash": {
			fullMethod:  "/",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
			wantErrMsg:  "invalid gRPC method format: /",
		},
		"empty": {
			fullMethod:  "",
			wantService: "",
			wantMethod:  "",
			wantErr:     true,
			wantErrMsg:  "invalid gRPC method format: ",
		},
		"empty service": {
			fullMethod:  "//method",
			wantService: "",
			wantMethod:  "method",
			wantErr:     true,
			wantErrMsg:  "invalid gRPC method: service name missing in //method",
		},
		"empty method": {
			fullMethod:  "/service/",
			wantService: "service",
			wantMethod:  "",
			wantErr:     true,
			wantErrMsg:  "invalid gRPC method: method name missing in /service/",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			svc, mthd, err := extractServiceMethod(tt.fullMethod)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrMsg != "" {
					assert.Equal(t, tt.wantErrMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantService, svc)
				assert.Equal(t, tt.wantMethod, mthd)
			}
		})
	}
}
