package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type byteSliceSource struct {
	data []byte
}

func (s *byteSliceSource) ReadAll() ([]byte, error) {
	return s.data, nil
}

func (s *byteSliceSource) WriteAll(p []byte) error {
	s.data = p
	return nil
}

func TestLureServiceExistsByURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		luresData string
		expected  bool
	}{
		{
			name:      "EmptyLures",
			input:     "/abc",
			luresData: "",
			expected:  false,
		},
		{
			name:  "OneLureInvalidURL",
			input: "/abc",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			expected: false,
		},
		{
			name:  "OneLureValidURL",
			input: "/he11o-lure",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			expected: true,
		},
		{
			name:  "TwoLureInvalidURL",
			input: "/he11o-lure3",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2`,
			expected: false,
		},
		{
			name:  "TwoLureValidURL",
			input: "/he11o-lure2",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2`,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewLureService(&byteSliceSource{[]byte(tt.luresData)})
			require.NoError(t, err)
			exists, err := s.ExistsByURL(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, exists)
		})
	}
}

func TestLureServiceGetAll(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		luresData string
		expected  []*APILure
	}{
		{
			name:      "EmptyLures",
			luresData: "",
			expected:  []*APILure{},
		},
		{
			name: "OneLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure",
					TargetURL: "https://www.example.com/some/url",
				},
			},
		},
		{
			name: "TwoLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2`,
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure1",
					TargetURL: "https://www.example.com/some/url1",
				},
				{
					Name:      "lure2",
					LureURL:   "/he11o-lure2",
					TargetURL: "https://www.example.com/some/url2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewLureService(&byteSliceSource{[]byte(tt.luresData)})
			require.NoError(t, err)
			lures, err := s.GetAll()
			require.NoError(t, err)
			sort.Slice(lures, func(i, j int) bool {
				return lures[i].Name < lures[j].Name
			})
			require.Equal(t, tt.expected, lures)
		})
	}
}

func TestLureServiceAdd(t *testing.T) {
	tests := []struct {
		name      string
		input     *APILure
		luresData string
		expected  []*APILure
	}{
		{
			name:      "EmptyLures",
			luresData: "",
			input: &APILure{
				Name:      "lure1",
				LureURL:   "/he11o-lure1",
				TargetURL: "https://www.example.com/some/url1",
			},
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure1",
					TargetURL: "https://www.example.com/some/url1",
				},
			},
		},
		{
			name: "OneLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			input: &APILure{
				Name:      "lure2",
				LureURL:   "/he11o-lure2",
				TargetURL: "https://www.example.com/some/url2",
			},
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure",
					TargetURL: "https://www.example.com/some/url",
				},
				{
					Name:      "lure2",
					LureURL:   "/he11o-lure2",
					TargetURL: "https://www.example.com/some/url2",
				},
			},
		},
		{
			name: "TwoLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2`,
			input: &APILure{
				Name:      "lure3",
				LureURL:   "/he11o-lure3",
				TargetURL: "https://www.example.com/some/url3",
			},
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure1",
					TargetURL: "https://www.example.com/some/url1",
				},
				{
					Name:      "lure2",
					LureURL:   "/he11o-lure2",
					TargetURL: "https://www.example.com/some/url2",
				},
				{
					Name:      "lure3",
					LureURL:   "/he11o-lure3",
					TargetURL: "https://www.example.com/some/url3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			byteSource := &byteSliceSource{[]byte(tt.luresData)}
			s, err := NewLureService(byteSource)
			require.NoError(t, err)

			err = s.Add(tt.input)
			require.NoError(t, err)

			lures, err := s.GetAll()
			require.NoError(t, err)
			require.Equal(t, tt.expected, lures)

			savedLures := parseLures(t, byteSource.data)
			require.Equal(t, tt.expected, savedLures)
		})
	}
}

func TestLureServiceAddInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input *APILure
	}{
		{
			name: "EmptyLureURL",
			input: &APILure{
				Name:      "lure1",
				TargetURL: "https://www.example.com/some/url1",
			},
		},
		{
			name: "EmptyTargetURL",
			input: &APILure{
				Name:    "lure1",
				LureURL: "/he11o-lure1",
			},
		},
		{
			name: "InvaludLureURL",
			input: &APILure{
				Name:      "lure1",
				LureURL:   "he11o-lure1",
				TargetURL: "https://www.example.com/some/url1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewLureService(&byteSliceSource{[]byte{}})
			require.NoError(t, err)

			err = s.Add(tt.input)
			require.Error(t, err)
		})
	}
}

func TestLureServiceDeleteByURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		luresData string
		expected  []*APILure
	}{
		{
			name: "OneLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			input:    "/he11o-lure",
			expected: []*APILure{},
		},
		{
			name: "TwoLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2`,
			input: "/he11o-lure2",
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure1",
					TargetURL: "https://www.example.com/some/url1",
				},
			},
		},
		{
			name: "ThreeLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2
            - name: lure3
              lure_url: /he11o-lure3
              target_url: https://www.example.com/some/url3`,
			input: "/he11o-lure2",
			expected: []*APILure{
				{
					Name:      "lure1",
					LureURL:   "/he11o-lure1",
					TargetURL: "https://www.example.com/some/url1",
				},
				{
					Name:      "lure3",
					LureURL:   "/he11o-lure3",
					TargetURL: "https://www.example.com/some/url3",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			byteSource := &byteSliceSource{[]byte(tt.luresData)}
			s, err := NewLureService(byteSource)
			require.NoError(t, err)

			err = s.DeleteByURL(tt.input)
			require.NoError(t, err)

			lures, err := s.GetAll()
			require.NoError(t, err)
			require.Equal(t, tt.expected, lures)

			savedLures := parseLures(t, byteSource.data)
			require.Equal(t, tt.expected, savedLures)
		})
	}
}

func parseLures(t *testing.T, data []byte) []*APILure {
	t.Helper()
	var doc struct {
		Lures []*APILure `yaml:"lures"`
	}
	err := yaml.Unmarshal(data, &doc)
	require.NoError(t, err)
	require.NoError(t, err)
	sort.Slice(doc.Lures, func(i, j int) bool {
		return doc.Lures[i].Name < doc.Lures[j].Name
	})
	return doc.Lures
}
