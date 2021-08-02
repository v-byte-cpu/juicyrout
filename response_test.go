package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResponseProcessorConvertCORS(t *testing.T) {
	tests := []struct {
		name   string
		origin string
	}{
		{
			name:   "WithProxyOrigin",
			origin: "www-google-com.example.com",
		},
		{
			name:   "WithTargetOrigin",
			origin: "www.google.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := NewResponseProcessor(NewDomainConverter("example.com")).(*responseProcessor)
			req := &http.Request{}
			req.Header = make(http.Header)
			req.Header["Origin"] = []string{tt.origin}
			resp := &http.Response{
				Request: req,
				Header:  make(http.Header),
			}
			resp.Header["Access-Control-Allow-Origin"] = []string{"www.google.com"}
			proc.convertCORS(resp)
			origins := resp.Header["Access-Control-Allow-Origin"]
			require.Equal(t, "www-google-com.example.com", origins[0])
		})
	}
}

func TestResponseProcessorConvertLocation(t *testing.T) {
	proc := NewResponseProcessor(NewDomainConverter("example.com")).(*responseProcessor)
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header["Location"] = []string{"https://www.google.com/abc"}
	resp.Header["Content-Location"] = []string{"https://www.google.com/doc.json"}
	proc.convertLocation(resp)
	locations := resp.Header["Location"]
	require.Equal(t, "https://www-google-com.example.com/abc", locations[0])
	locations = resp.Header["Content-Location"]
	require.Equal(t, "https://www-google-com.example.com/doc.json", locations[0])
}

func TestResponseProcessorConvertRelativeLocation(t *testing.T) {
	proc := NewResponseProcessor(NewDomainConverter("example.com")).(*responseProcessor)
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header["Location"] = []string{"/abc"}
	resp.Header["Content-Location"] = []string{"/doc.json"}
	proc.convertLocation(resp)
	locations := resp.Header["Location"]
	require.Equal(t, "/abc", locations[0])
	locations = resp.Header["Content-Location"]
	require.Equal(t, "/doc.json", locations[0])
}

func TestResponseProcessorWriteCookies(t *testing.T) {
	proc := NewResponseProcessor(NewDomainConverter("example.com")).(*responseProcessor)
	w := httptest.NewRecorder()
	resp := &http.Response{
		Header: make(http.Header),
	}
	cookie := &http.Cookie{
		Name:     "sessionID",
		Value:    "abc",
		Domain:   "www.google.com",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	resp.Header.Add("Set-Cookie", cookie.String())
	proc.writeCookies(w, resp)

	result := w.Result()
	cookies := result.Cookies()
	require.NotZero(t, len(cookies))
	require.Equal(t, "sessionID", cookies[0].Name)
	require.Equal(t, "abc", cookies[0].Value)
	require.Equal(t, "www-google-com.example.com", cookies[0].Domain)
	require.True(t, cookies[0].Secure)
	require.False(t, cookies[0].HttpOnly)
	require.Equal(t, http.SameSiteNoneMode, cookies[0].SameSite)

	require.Zero(t, len(resp.Header["Set-Cookie"]))
}

func TestResponseProcessorWriteHeaders(t *testing.T) {
	proc := NewResponseProcessor(NewDomainConverter("example.com")).(*responseProcessor)
	w := httptest.NewRecorder()
	resp := &http.Response{
		Header: make(http.Header),
	}
	resp.Header.Add("Content-Type", "application/javascript")
	resp.Header.Add("Content-Encoding", "gzip")

	proc.writeHeaders(w, resp)

	result := w.Result()
	require.Equal(t, "application/javascript", result.Header["Content-Type"][0])
	require.Equal(t, "gzip", result.Header["Content-Encoding"][0])
}

func TestResponseProcessorWriteBody(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "WithoutURLs",
			input: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
			expected: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
		},
		{
			name: "WithURLs",
			input: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  <link rel="dns-prefetch" href="https://github.githubassets.com">
	  <link rel="dns-prefetch" href="http://avatars.githubusercontent.com">
	  <link rel="dns-prefetch" href="https://github-cloud.s3.amazonaws.com">
	  <link rel="dns-prefetch" href="//user-images.githubusercontent.com/">
	  <link rel="mask-icon" href="https://github.githubassets.com/pinned-octocat.svg" color="#000000">
	  <link rel="shortcut icon" type="image/x-icon" href=//limg.test.com/re/i/meta/favicon.ico />
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
			expected: `
	<!DOCTYPE html>
	<html lang="en">
	  <head>
		<meta charset="utf-8">
	  <link rel="dns-prefetch" href="https://github-githubassets-com.example.com">
	  <link rel="dns-prefetch" href="http://avatars-githubusercontent-com.example.com">
	  <link rel="dns-prefetch" href="https://github--cloud-s3-amazonaws-com.example.com">
	  <link rel="dns-prefetch" href="//user--images-githubusercontent-com.example.com/">
	  <link rel="mask-icon" href="https://github-githubassets-com.example.com/pinned-octocat.svg" color="#000000">
	  <link rel="shortcut icon" type="image/x-icon" href=//limg-test-com.example.com/re/i/meta/favicon.ico />
	  </head>
	  <body>
	  	Hello!
	  </body>
	</html>
	`,
		},
		{
			name:     "LargeBuffer",
			input:    strings.Repeat(`<link rel="dns-prefetch" href="https://github.githubassets.com">`, 4096),
			expected: strings.Repeat(`<link rel="dns-prefetch" href="https://github-githubassets-com.example.com">`, 4096),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc := NewResponseProcessor(NewDomainConverter("example.com")).(*responseProcessor)
			w := httptest.NewRecorder()

			resp := &http.Response{
				Body: ioutil.NopCloser(strings.NewReader(tt.input)),
			}

			proc.writeBody(w, resp)
			result := w.Result()
			data, err := io.ReadAll(result.Body)
			require.NoError(t, err)
			assert.Equal(t, len(tt.expected), len(data))
			require.Equal(t, tt.expected, string(data))
		})
	}
}
