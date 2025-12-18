package grpc

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	tests := map[string]struct {
		prefix  string
		env     map[string]string
		want    Config
		wantErr bool
	}{
		"defaults": {
			prefix: "test",
			env:    map[string]string{},
			want: Config{
				ShutdownTimeout: 20 * time.Second,
				APIHost:         "0.0.0.0:50051",
				DebugHost:       "0.0.0.0:3010",
				MetricsHost:     "0.0.0.0:2112",
				Build:           "dev",
				Desc:            "example grpc server",
				Namespace:       "test",
				Version:         "test",
				Name:            "test",
			},
		},
		"env vars set": {
			prefix: "test",
			env: map[string]string{
				"TEST_APIHOST": "1.2.3.4:5678",
				"TEST_NAME":    "custom-name",
			},
			want: Config{
				ShutdownTimeout: 20 * time.Second,
				APIHost:         "1.2.3.4:5678",
				DebugHost:       "0.0.0.0:3010",
				MetricsHost:     "0.0.0.0:2112",
				Build:           "dev",
				Desc:            "example grpc server",
				Namespace:       "test",
				Version:         "test",
				Name:            "custom-name",
			},
		},
		"explicit namespace": {
			prefix: "test",
			env: map[string]string{
				"TEST_NAMESPACE": "custom-ns",
			},
			want: Config{
				ShutdownTimeout: 20 * time.Second,
				APIHost:         "0.0.0.0:50051",
				DebugHost:       "0.0.0.0:3010",
				MetricsHost:     "0.0.0.0:2112",
				Build:           "dev",
				Desc:            "example grpc server",
				Namespace:       "custom-ns",
				Version:         "test",
				Name:            "test",
			},
		},
		"invalid duration": {
			prefix: "test",
			env: map[string]string{
				"TEST_SHUTDOWNTIMEOUT": "invalid",
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got, err := LoadConfig(tt.prefix)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
