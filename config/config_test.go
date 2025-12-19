package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	Val       string `default:"default"`
	IntVal    int    `default:"0"`
	Namespace string
}

func TestLoadConfig(t *testing.T) {
	tests := map[string]struct {
		prefix  string
		env     map[string]string
		want    TestConfig
		wantErr bool
	}{
		"defaults": {
			prefix: "TEST",
			env:    map[string]string{},
			want: TestConfig{
				Val:       "default",
				IntVal:    0,
				Namespace: "TEST",
			},
		},
		"env var set": {
			prefix: "TEST",
			env: map[string]string{
				"TEST_VAL": "custom",
			},
			want: TestConfig{
				Val:       "custom",
				IntVal:    0,
				Namespace: "TEST",
			},
		},
		"namespace override": {
			prefix: "TEST",
			env: map[string]string{
				"TEST_NAMESPACE": "custom_ns",
			},
			want: TestConfig{
				Val:       "default",
				IntVal:    0,
				Namespace: "custom_ns",
			},
		},
		"invalid int": {
			prefix: "TEST",
			env: map[string]string{
				"TEST_INTVAL": "invalid",
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

			got, err := LoadConfig[TestConfig](tt.prefix)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
