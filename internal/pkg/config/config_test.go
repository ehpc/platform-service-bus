package config

import (
	"github.com/google/go-cmp/cmp"
	"platform-service-bus/internal/pkg/adapter"
	"testing"
)

func TestLoad(t *testing.T) {
	table := []struct {
		name          string
		input         string
		expected      Config
		expectedError bool
	}{
		{
			name:  "Adapters",
			input: `{"adapters":[{"port":7000},{"port":7001}]}`,
			expected: Config{
				Adapters: []adapter.Adapter{
					adapter.Adapter{
						Port: 7000,
					},
					adapter.Adapter{
						Port: 7001,
					},
				},
			},
			expectedError: false,
		},
		{
			name:          "Wrong JSON",
			input:         `{adapters:[]}`,
			expectedError: true,
		},
	}
	for _, item := range table {
		t.Run(item.name, func(t *testing.T) {
			got, err := Load("", fileReader(func(filename string) ([]byte, error) {
				return []byte(item.input), nil
			}))
			if item.expectedError && err == nil {
				t.Errorf("Expected an error, got %v", got)
			} else if !cmp.Equal(got, item.expected) {
				t.Errorf("Expected %v, got %v, err %v", item.expected, got, err)
			}
		})
	}
}
