package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainCoverterToProxyDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "TargetDomain",
			input:    "www.google.com",
			expected: "www-google-com.example.com",
		},
		{
			name:     "TargetDomainWithSlash",
			input:    "static-content.google.com",
			expected: "static--content-google-com.example.com",
		},
		{
			name:     "ProxyDomain",
			input:    "www-google-com.example.com",
			expected: "www-google-com.example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewDomainConverter("example.com")
			result := conv.ToProxyDomain(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainCoverterToTargetDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ProxyDomain",
			input:    "www-google-com.example.com",
			expected: "www.google.com",
		},
		{
			name:     "ProxyDomainWithSlash",
			input:    "static--content-google-com.example.com",
			expected: "static-content.google.com",
		},
		{
			name:     "TargetDomain",
			input:    "www.google.com",
			expected: "www.google.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewDomainConverter("example.com")
			result := conv.ToTargetDomain(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainCoverterToTargetURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ProxyDomain",
			input:    "https://www-google-com.example.com",
			expected: "https://www.google.com",
		},
		{
			name:     "ProxyDomainWithSlash",
			input:    "https://static--content-google-com.example.com",
			expected: "https://static-content.google.com",
		},
		{
			name:     "ProxyDomainWithPath",
			input:    "https://www-google-com.example.com/abc",
			expected: "https://www.google.com/abc",
		},
		{
			name:     "TargetDomain",
			input:    "https://www.google.com",
			expected: "https://www.google.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewDomainConverter("example.com")
			result := conv.ToTargetURL(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainCoverterStaticMappingDomain(t *testing.T) {
	conv := NewDomainConverter("example.com")
	conv.AddStaticMapping("www.example.com", "static.google.com")

	result := conv.ToTargetDomain("www-google-com.example.com")
	require.Equal(t, "www.google.com", result)
	result = conv.ToProxyDomain("www.google.com")
	require.Equal(t, "www-google-com.example.com", result)

	require.Equal(t, "static.google.com", conv.ToTargetDomain("www.example.com"))
	require.Equal(t, "www.example.com", conv.ToProxyDomain("static.google.com"))
}
