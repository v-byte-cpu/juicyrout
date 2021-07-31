package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestProcessor(t *testing.T) {
	req, err := http.NewRequest("GET", "https://www-google-com.example.com", nil)
	require.NoError(t, err)
	req.Header["Referer"] = []string{"https://abc.com"}
	req.Header["Origin"] = []string{"https://www-google-com.example.com"}

	p := NewRequestProcessor(NewDomainConverter("example.com"))
	result := p.Process(req)
	require.Equal(t, "www.google.com", result.URL.Host)
	require.Zero(t, len(req.Header["Referer"]))
	require.Equal(t, []string{"https://www.google.com"}, result.Header["Origin"])
}
