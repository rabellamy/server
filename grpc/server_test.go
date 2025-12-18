package grpc

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestNewServer(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config  Config
		wantErr bool
	}{
		"valid config": {
			config: Config{
				Namespace: "test_server",
				APIHost:   "localhost:0",
			},
			wantErr: false,
		},
		"invalid namespace": {
			config: Config{
				Namespace: "123invalid",
			},
			wantErr: true,
		},
		"register fail": {
			config: Config{
				Namespace: "test_server_collision",
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if name == "register fail" {
				// Manually trigger a collision by creating a server with the same namespace first
				logger := slog.New(slog.NewTextHandler(io.Discard, nil))
				_, err := NewServer(context.Background(), tt.config, nil, logger)
				assert.NoError(t, err)
			}

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			got, err := NewServer(context.Background(), tt.config, nil, logger)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestRun(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		config     Config
		cancelCtx  bool
		sendSignal bool
		preCancel  bool
		wantErr    bool
	}{
		"context cancellation shutdown": {
			config: Config{
				Namespace:       "test_run_ctx",
				APIHost:         "localhost:0",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
			cancelCtx: true,
		},
		"signal shutdown": {
			config: Config{
				Namespace:       "test_run_signal",
				APIHost:         "localhost:0",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
			sendSignal: true,
		},
		"context pre-cancelled": {
			config: Config{
				Namespace:       "test_run_pre_cancel",
				APIHost:         "localhost:0",
				MetricsHost:     "localhost:0",
				ShutdownTimeout: 5 * time.Second,
			},
			preCancel: true,
		},
		"invalid api host": {
			config: Config{
				Namespace:   "test_run_invalid_api",
				APIHost:     "invalid-host:port",
				MetricsHost: "localhost:0",
			},
			wantErr: true,
		},
		"invalid metrics host": {
			config: Config{
				Namespace:   "test_run_invalid_metrics",
				APIHost:     "localhost:0",
				MetricsHost: "invalid-host:port",
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.preCancel {
				cancel()
			}

			server, err := NewServer(ctx, tt.config, nil, logger)
			assert.NoError(t, err)

			shutdownChan := make(chan os.Signal, 1)
			errChan := make(chan error, 1)
			go func() {
				errChan <- server.run(shutdownChan)
			}()

			// Give server time to start
			time.Sleep(50 * time.Millisecond)

			if tt.sendSignal {
				// Send mock signal directly to the channel
				shutdownChan <- os.Interrupt
			} else if tt.cancelCtx {
				cancel()
			}

			// We expect no error for successful runs/shutdowns
			err = <-errChan
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRegister(t *testing.T) {
	t.Parallel()

	config := Config{
		Namespace: "test_register",
		APIHost:   "localhost:0",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := context.Background()

	called := false
	register := func(s *grpc.Server) {
		called = true
	}

	_, err := NewServer(ctx, config, register, logger)
	assert.NoError(t, err)
	assert.True(t, called, "register function should have been called")
}

func TestHealthCheck(t *testing.T) {
	tests := map[string]struct {
		service string
		want    grpc_health_v1.HealthCheckResponse_ServingStatus
		wantErr bool
	}{
		"overall server is serving": {
			service: "",
			want:    grpc_health_v1.HealthCheckResponse_SERVING,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {

			// Find a random free port
			lis, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)
			addr := lis.Addr().String()
			lis.Close()

			// Sanitize namespace for Prometheus
			ns := "test_health_" + strings.ReplaceAll(name, " ", "_")
			config := Config{
				Namespace:       ns,
				APIHost:         addr,
				MetricsHost:     "127.0.0.1:0",
				ShutdownTimeout: 5 * time.Second,
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			server, err := NewServer(ctx, config, nil, logger)
			if !assert.NoError(t, err) {
				return
			}

			errChan := make(chan error, 1)
			go func() {
				errChan <- server.Run()
			}()

			// Give server time to start
			time.Sleep(100 * time.Millisecond)

			// Check if server failed to start
			select {
			case err := <-errChan:
				t.Fatalf("server failed to start: %v", err)
			default:
			}

			// Connect client
			conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			assert.NoError(t, err)
			defer conn.Close()

			client := grpc_health_v1.NewHealthClient(conn)

			// Retry check a few times
			var resp *grpc_health_v1.HealthCheckResponse
			resp, err = client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{
				Service: tt.service,
			})

			time.Sleep(100 * time.Millisecond)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				if assert.NoError(t, err) && resp != nil {
					assert.Equal(t, tt.want, resp.Status)
				}
			}

			cancel()
			err = <-errChan
			assert.NoError(t, err)
		})
	}
}

func TestShutdownServers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		ctxTimeout time.Duration
		wantErr    bool
		wantErrMsg string
		signal     os.Signal
		slowGrace  bool
	}{
		"successful shutdown": {
			ctxTimeout: 5 * time.Second,
			wantErr:    false,
		},
		"metrics shutdown failure": {
			ctxTimeout: 0, // Pre-cancelled context
			wantErr:    true,
			wantErrMsg: "metrics server could not stop gracefully",
		},
		"grpc shutdown timeout": {
			ctxTimeout: 50 * time.Millisecond,
			wantErr:    true,
			wantErrMsg: "grpc server shutdown timed out",
			slowGrace:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Sanitize namespace for Prometheus
			ns := "test_shutdown_" + strings.ReplaceAll(name, " ", "_")
			config := Config{
				Namespace:       ns,
				APIHost:         "127.0.0.1:0",
				MetricsHost:     "127.0.0.1:0",
				ShutdownTimeout: tt.ctxTimeout,
			}
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))

			blockChan := make(chan struct{})
			var server *Server
			var err error

			if tt.slowGrace {
				// Register a blocking interceptor to simulate slow gRPC GracefulStop
				blockInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
					select {
					case <-blockChan:
					case <-ctx.Done():
						return nil, ctx.Err()
					}
					return handler(ctx, req)
				}
				server, err = NewServer(context.Background(), config, nil, logger, grpc.UnaryInterceptor(blockInterceptor))
			} else {
				server, err = NewServer(context.Background(), config, nil, logger)
			}
			assert.NoError(t, err)

			if name == "metrics shutdown failure" {
				// Replace metrics handler with one we can block
				server.metricsServer.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					select {
					case <-blockChan:
					case <-r.Context().Done():
					}
					w.WriteHeader(http.StatusOK)
				})
			}

			if tt.slowGrace {
				lis, err := net.Listen("tcp", "127.0.0.1:0")
				assert.NoError(t, err)
				addr := lis.Addr().String()
				go server.grpcServer.Serve(lis)

				conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
				assert.NoError(t, err)
				defer conn.Close()

				client := grpc_health_v1.NewHealthClient(conn)
				go func() {
					_, _ = client.Check(context.Background(), &grpc_health_v1.HealthCheckRequest{Service: ""})
				}()
				time.Sleep(100 * time.Millisecond)
			}

			if name == "metrics shutdown failure" {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				assert.NoError(t, err)
				go server.metricsServer.Serve(ln)

				// Make a request that will block
				go func() {
					_, _ = http.Get("http://" + ln.Addr().String())
				}()
				time.Sleep(100 * time.Millisecond)
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.ctxTimeout)
			defer cancel()

			err = server.shutdownServers(ctx, tt.signal)

			// Unblock everything
			select {
			case <-blockChan:
			default:
				close(blockChan)
			}

			if tt.wantErr {
				if assert.Error(t, err) && tt.wantErrMsg != "" {
					assert.Contains(t, err.Error(), tt.wantErrMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
