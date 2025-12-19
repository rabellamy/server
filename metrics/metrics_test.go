package metrics

import (
	"testing"

	"github.com/rabellamy/promstrap/strategy"
	"github.com/stretchr/testify/assert"
)

func TestNewRED(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		namespace      string
		requestLabels  []string
		durationLabels []string
		want           *strategy.RED
		wantErr        bool
	}{
		"valid namespace": {
			namespace:      "test_metrics",
			requestLabels:  []string{"path", "verb"},
			durationLabels: []string{"path"},
			want:           &strategy.RED{},
			wantErr:        false,
		},
		"empty namespace": {
			namespace:      "",
			requestLabels:  []string{"path", "verb"},
			durationLabels: []string{"path"},
			want:           nil,
			wantErr:        true,
		},
		"invalid namespace": {
			namespace:      "123invalid",
			requestLabels:  []string{"path", "verb"},
			durationLabels: []string{"path"},
			want:           &strategy.RED{},
			wantErr:        true,
		},
		"missing duration labels": {
			namespace:      "valid_namespace",
			requestLabels:  []string{"path", "verb"},
			durationLabels: nil,
			want:           nil,
			wantErr:        true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := NewRED(tt.namespace, "http", tt.requestLabels, tt.durationLabels)

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
