package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDomainCoverterToProxy(t *testing.T) {
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
			result := conv.ToProxy(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainCoverterToProxyCookie(t *testing.T) {
	tests := []struct {
		name       string
		baseDomain string
		input      string
		expected   string
	}{
		{
			name:       "TargetDomain",
			input:      "www.google.com",
			baseDomain: "example.com",
			expected:   "www-google-com.example.com",
		},
		{
			name:       "TargetDomainWithSlash",
			input:      "static-content.google.com",
			baseDomain: "example.com",
			expected:   "static--content-google-com.example.com",
		},
		{
			name:       "ProxyDomain",
			input:      "www-google-com.example.com",
			baseDomain: "example.com",
			expected:   "www-google-com.example.com",
		},
		{
			name:       "TargetDomainWithDot",
			input:      ".www.google.com",
			baseDomain: "example.com",
			expected:   "www-google-com.example.com",
		},
		{
			name:       "BaseDomainWithPort",
			input:      ".www.google.com",
			baseDomain: "example.com:8091",
			expected:   "www-google-com.example.com",
		},
		{
			name:       "EmptyDomain",
			input:      "",
			baseDomain: "example.com:8091",
			expected:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewDomainConverter(tt.baseDomain)
			result := conv.ToProxyCookie(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainCoverterToTarget(t *testing.T) {
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
			result := conv.ToTarget(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainCoverterStaticMapping(t *testing.T) {
	conv := NewDomainConverter("example.com")
	conv.AddStaticMapping("www.example.com", "static.google.com")

	result := conv.ToTarget("www-google-com.example.com")
	require.Equal(t, "www.google.com", result)
	result = conv.ToProxy("www.google.com")
	require.Equal(t, "www-google-com.example.com", result)

	require.Equal(t, "static.google.com", conv.ToTarget("www.example.com"))
	require.Equal(t, "www.example.com", conv.ToProxy("static.google.com"))
}

func TestDomainCoverterStaticMappingCookie(t *testing.T) {
	conv := NewDomainConverter("example.com")
	conv.AddStaticMapping("www.example.com", "static.google.com")
	conv.AddStaticMapping("abc.example.com:8091", "abc.google.com")

	result := conv.ToProxyCookie("www.google.com")
	require.Equal(t, "www-google-com.example.com", result)

	require.Equal(t, "www.example.com", conv.ToProxyCookie("static.google.com"))
	require.Equal(t, "abc.example.com", conv.ToProxyCookie("abc.google.com"))
}
