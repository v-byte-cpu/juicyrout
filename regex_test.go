package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestURLRegexProcessorProcessAll(t *testing.T) {
	var buff bytes.Buffer
	conv := NewDomainConverter("example.com")
	urlProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	err := urlProc.ProcessAll(&buff, strings.NewReader("q=https://google-com.example.com"))
	require.NoError(t, err)
	require.Equal(t, "q=https://google.com", utils.UnsafeString(buff.Bytes()))
}

//nolint:errcheck
func BenchmarkURLRegexProcessorProcessAll(b *testing.B) {
	conv := NewDomainConverter("example.com")
	proc := newURLRegexProcessor(func(domain string) string {
		return conv.ToProxyDomain(domain)
	})
	r := strings.NewReader(`<link rel="dns-prefetch" href="https://github.githubassets.com">`)
	for i := 0; i < b.N; i++ {
		proc.ProcessAll(io.Discard, r)
		r.Seek(0, io.SeekStart)
	}
}
