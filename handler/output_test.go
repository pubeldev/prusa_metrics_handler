package handler

import (
	"context"
	"testing"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/stretchr/testify/assert"
)

type mockInfluxClient struct {
	writeFunc func(ctx context.Context, data []byte, opts ...influxdb3.WriteOption) error
}

func (m *mockInfluxClient) Write(ctx context.Context, data []byte, opts ...influxdb3.WriteOption) error {
	if m.writeFunc != nil {
		return m.writeFunc(ctx, data, opts...)
	}
	return nil
}

func TestSentToInflux(t *testing.T) {
	tests := []struct {
		name        string
		message     []string
		writeFunc   func(ctx context.Context, data []byte, opts ...influxdb3.WriteOption) error
		expected    bool
		expectError bool
	}{
		{
			name:    "successful write",
			message: []string{"line1", "line2"},
			writeFunc: func(ctx context.Context, data []byte, opts ...influxdb3.WriteOption) error {
				return nil
			},
			expected:    false,
			expectError: false,
		},
		{
			name:    "write error",
			message: []string{"line1", "line2"},
			writeFunc: func(ctx context.Context, data []byte, opts ...influxdb3.WriteOption) error {
				return assert.AnError
			},
			expected:    false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockInfluxClient{
				writeFunc: tt.writeFunc,
			}

			result, err := sentToInflux(tt.message, mockClient)
			assert.Equal(t, tt.expected, result)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
