package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStaticLureService(t *testing.T) {
	tests := []struct {
		name     string
		lureURLs []string
		input    string
		expected bool
	}{
		{
			name:     "EmptyLures",
			lureURLs: nil,
			input:    "/abc",
			expected: false,
		},
		{
			name:     "OneLureInvalidURL",
			lureURLs: []string{"/he11o_lure"},
			input:    "/abc",
			expected: false,
		},
		{
			name:     "OneLureValidURL",
			lureURLs: []string{"/he11o_lure"},
			input:    "/he11o_lure",
			expected: true,
		},
		{
			name:     "TwoLureInvalidURL",
			lureURLs: []string{"/he11o_lure1", "/he11o_lure2"},
			input:    "/he11o_lure3",
			expected: false,
		},
		{
			name:     "TwoLureValidURL",
			lureURLs: []string{"/he11o_lure1", "/he11o_lure2"},
			input:    "/he11o_lure2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewStaticLureService(tt.lureURLs)
			exists, err := s.ExistsByURL(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, exists)
		})
	}
}
