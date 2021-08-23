package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2/utils"
	"github.com/stretchr/testify/require"
)

func TestURLRegex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://google.com",
			expected: "//google.com",
		},
		{
			input:    "https://GoOgLe.CoM",
			expected: "//GoOgLe.CoM",
		},
		{
			input:    "https://static.google.com",
			expected: "//static.google.com",
		},
		{
			input:    "https://static-content.google.com",
			expected: "//static-content.google.com",
		},
		{
			input:    "https://static--content-google-com.example.com",
			expected: "//static--content-google-com.example.com",
		},
	}
	for _, tt := range tests {
		data := urlRegexp.FindString(tt.input)
		require.Equal(t, tt.expected, data)
	}
}

func TestURLRegexProcessorProcessAll(t *testing.T) {
	var buff bytes.Buffer
	conv := NewDomainConverter("example.com")
	urlProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	err := urlProc.ProcessAll(&buff, strings.NewReader("q=https://static--content-google-com.example.com"))
	require.NoError(t, err)
	require.Equal(t, "q=https://static-content.google.com", utils.UnsafeString(buff.Bytes()))
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

func TestReplaceRegexReaderRead(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		buff        []byte
		expected    string
		expectedErr error
	}{
		{
			name:        "EmptyBuffer",
			input:       "abc",
			buff:        nil,
			expected:    "",
			expectedErr: nil,
		},
		{
			name:        "BufferEqualToInputSize",
			input:       "abc",
			buff:        make([]byte, 3),
			expected:    "abc",
			expectedErr: nil,
		},
		{
			name:        "BufferLessThanInputSize",
			input:       "abcdef",
			buff:        make([]byte, 3),
			expected:    "abc",
			expectedErr: nil,
		},
		{
			name:        "BufferMoreThanInputSize",
			input:       "abcdef",
			buff:        make([]byte, 256),
			expected:    "abcdef",
			expectedErr: io.EOF,
		},
		{
			name:        "InputWithRegex",
			input:       `<link rel="dns-prefetch" href="https://github--cloud-s3-amazonaws-com.example.com">`,
			buff:        make([]byte, 256),
			expected:    `<link rel="dns-prefetch" href="https://github-cloud.s3.amazonaws.com">`,
			expectedErr: io.EOF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conv := NewDomainConverter("example.com")
			proc := newURLRegexProcessor(func(domain string) string {
				return conv.ToTargetDomain(domain)
			})
			r := NewReplaceRegexReader(strings.NewReader(tt.input), proc)

			n, err := r.Read(tt.buff)
			require.Equal(t, tt.expected, string(tt.buff[:n]))
			require.Equal(t, len(tt.expected), n)
			if tt.expectedErr == io.EOF {
				if err == nil {
					n, err = r.Read(tt.buff)
					require.Zero(t, n)
				}
				require.Equal(t, io.EOF, err)
			}
		})
	}
}

func TestReplaceRegexReaderReadLargeInput(t *testing.T) {
	conv := NewDomainConverter("example.com")
	proc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	input := strings.Repeat(`<link rel="dns-prefetch" href="https://github-githubassets-com.example.com">`, 4096)
	expected := strings.Repeat(`<link rel="dns-prefetch" href="https://github.githubassets.com">`, 4096)
	r := NewReplaceRegexReader(strings.NewReader(input), proc)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, len(expected), len(data))
	require.Equal(t, expected, string(data))
}

func TestReplaceRegexReaderReadByParts(t *testing.T) {
	conv := NewDomainConverter("example.com")
	proc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	input := strings.Repeat(`<link rel="dns-prefetch" href="https://github-githubassets-com.example.com">`, 256)
	output := strings.Repeat(`<link rel="dns-prefetch" href="https://github.githubassets.com">`, 256)

	tests := []struct {
		name   string
		chunks []int
	}{
		{
			name:   "test1",
			chunks: []int{37, len(output) - 37},
		},
		{
			name:   "test2",
			chunks: []int{127, 56, len(output) - 127 - 56},
		},
		{
			name:   "test3",
			chunks: []int{261, 23, 1264, len(output) - 261 - 23 - 1264},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewReplaceRegexReader(strings.NewReader(input), proc)
			offset := 0
			for i := 0; i < len(tt.chunks); i++ {
				buff := make([]byte, tt.chunks[i])
				n, err := r.Read(buff)
				if err == io.EOF {
					require.Equal(t, len(tt.chunks)-1, i)
				} else {
					require.NoError(t, err)
				}
				require.Equal(t, output[offset:offset+len(buff)], string(buff))
				offset += n
			}
		})
	}
}

func TestReplaceRegexReaderClose(t *testing.T) {
	conv := NewDomainConverter("example.com")
	proc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	r := NewReplaceRegexReader(strings.NewReader("abc"), proc)
	for i := 0; i < 2; i++ {
		err := r.Close()
		require.NoError(t, err)
		n, err := r.Read(make([]byte, 10))
		require.Equal(t, io.EOF, err)
		require.Zero(t, n)
	}
}
