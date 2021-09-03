package main

import (
	"net/http"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
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

func TestLureServiceGetByURL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		luresData string
		expected  *APILure
	}{
		{
			name:      "EmptyLures",
			luresData: "",
			input:     "/lure/url",
			expected:  nil,
		},
		{
			name: "OneLureURL",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			input: "/he11o-lure",
			expected: &APILure{
				Name:      "lure1",
				LureURL:   "/he11o-lure",
				TargetURL: "https://www.example.com/some/url",
			},
		},
		{
			name: "OneLureURLNonExisting",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure
              target_url: https://www.example.com/some/url`,
			input:    "/he11o-lure2",
			expected: nil,
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
			expected: &APILure{
				Name:      "lure2",
				LureURL:   "/he11o-lure2",
				TargetURL: "https://www.example.com/some/url2",
			},
		},
		{
			name: "TwoLureURLNonExisting",
			luresData: `
            lures:
            - name: lure1
              lure_url: /he11o-lure1
              target_url: https://www.example.com/some/url1
            - name: lure2
              lure_url: /he11o-lure2
              target_url: https://www.example.com/some/url2`,
			input:    "/he11o-lure3",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewLureService(&byteSliceSource{[]byte(tt.luresData)})
			require.NoError(t, err)
			lure, err := s.GetByURL(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, lure)
		})
	}
}

func TestLureServiceGetAll(t *testing.T) {
	tests := []struct {
		name      string
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

func TestNewSessionCookie(t *testing.T) {
	tests := []struct {
		name     string
		destURL  *url.URL
		cookie   *http.Cookie
		expected *SessionCookie
	}{
		{
			name:    "CookieWithExpires",
			destURL: &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
			cookie: &http.Cookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "google.com",
				Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			},
			expected: &SessionCookie{
				Name:           "Cookie1",
				Value:          "Value1",
				Path:           "/",
				Domain:         "google.com",
				ExpirationDate: 1643767322.123123,
				HTTPOnly:       true,
				Secure:         true,
				SameSite:       "no_restriction",
				Session:        false,
			},
		},
		{
			name:    "CookieWithEmptyExpiration",
			destURL: &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
			cookie: &http.Cookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "google.com",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			},
			expected: &SessionCookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "google.com",
				HTTPOnly: true,
				Secure:   true,
				SameSite: "no_restriction",
				Session:  true,
			},
		},
		{
			name:    "CookieWithEmptyPath",
			destURL: &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
			cookie: &http.Cookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Domain:   "google.com",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			},
			expected: &SessionCookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "google.com",
				HTTPOnly: true,
				Secure:   true,
				SameSite: "no_restriction",
				Session:  true,
			},
		},
		{
			name:    "CookieWithDotDomain",
			destURL: &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
			cookie: &http.Cookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Domain:   ".google.com",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			},
			expected: &SessionCookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "google.com",
				HTTPOnly: true,
				Secure:   true,
				SameSite: "no_restriction",
				Session:  true,
			},
		},
		{
			name:    "CookieWithEmptyDomain",
			destURL: &url.URL{Scheme: "https", Host: "www.google.com", Path: "/"},
			cookie: &http.Cookie{
				Name:     "Cookie1",
				Value:    "Value1",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			},
			expected: &SessionCookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "www.google.com",
				HTTPOnly: true,
				Secure:   true,
				SameSite: "no_restriction",
				Session:  true,
			},
		},
		{
			name:    "CookieWithEmptyDomainWithPort",
			destURL: &url.URL{Scheme: "https", Host: "www.google.com:8090", Path: "/"},
			cookie: &http.Cookie{
				Name:     "Cookie1",
				Value:    "Value1",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteNoneMode,
			},
			expected: &SessionCookie{
				Name:     "Cookie1",
				Value:    "Value1",
				Path:     "/",
				Domain:   "www.google.com",
				HTTPOnly: true,
				Secure:   true,
				SameSite: "no_restriction",
				Session:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := newSessionCookie(tt.destURL, tt.cookie)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGetDomainCookieNamesRegexp(t *testing.T) {
	type subtest struct {
		domain         string
		cookieName     string
		containsDomain bool
		matches        bool
	}
	tests := []struct {
		name     string
		cookies  []*SessionCookieConfig
		subtests []subtest
	}{
		{
			name: "OneDomainOneCookie",
			cookies: []*SessionCookieConfig{
				{
					Name:   "sessionid",
					Domain: "example.com",
				},
			},
			subtests: []subtest{
				{
					domain:         "example.com",
					cookieName:     "sessionid",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example.com",
					cookieName:     "sessionid2",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example.com",
					cookieName:     "2sessionid",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example2.com",
					cookieName:     "sessionid",
					containsDomain: false,
				},
			},
		},
		{
			name: "OneDomainOneCookieWithDot",
			cookies: []*SessionCookieConfig{
				{
					Name:   "sessionid",
					Domain: ".example.com",
				},
			},
			subtests: []subtest{
				{
					domain:         "example.com",
					cookieName:     "sessionid",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example.com",
					cookieName:     "sessionid2",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example.com",
					cookieName:     "2sessionid",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example2.com",
					cookieName:     "sessionid",
					containsDomain: false,
				},
			},
		},
		{
			name: "OneDomainTwoCookies",
			cookies: []*SessionCookieConfig{
				{
					Name:   "sessionid",
					Domain: "example.com",
				},
				{
					Name:   "sessionid3",
					Domain: "example.com",
				},
			},
			subtests: []subtest{
				{
					domain:         "example.com",
					cookieName:     "sessionid",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example.com",
					cookieName:     "sessionid2",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example.com",
					cookieName:     "2sessionid",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example2.com",
					cookieName:     "sessionid",
					containsDomain: false,
				},
				{
					domain:         "example.com",
					cookieName:     "sessionid3",
					containsDomain: true,
					matches:        true,
				},
			},
		},
		{
			name: "OneDomainTwoCookiesWithRegexp",
			cookies: []*SessionCookieConfig{
				{
					Name:   ".*",
					Domain: "example.com",
				},
				{
					Name:   "abc.*",
					Domain: "example.com",
					Regexp: true,
				},
			},
			subtests: []subtest{
				{
					domain:         "example.com",
					cookieName:     ".*",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example.com",
					cookieName:     "sessionid2",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example.com",
					cookieName:     "2sessionid",
					containsDomain: true,
					matches:        false,
				},
				{
					domain:         "example2.com",
					cookieName:     "sessionid",
					containsDomain: false,
				},
				{
					domain:         "example.com",
					cookieName:     "abc123",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example.com",
					cookieName:     "1abc123",
					containsDomain: true,
					matches:        false,
				},
			},
		},
		{
			name: "OneDomainTwoCookiesWithRegexp2",
			cookies: []*SessionCookieConfig{
				{
					Name:   ".*",
					Domain: "example.com",
					Regexp: true,
				},
				{
					Name:   "sessionid",
					Domain: "example.com",
					Regexp: true,
				},
			},
			subtests: []subtest{
				{
					domain:         "example.com",
					cookieName:     ".*",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example.com",
					cookieName:     "mid",
					containsDomain: true,
					matches:        true,
				},
				{
					domain:         "example2.com",
					cookieName:     "sessionid",
					containsDomain: false,
				},
				{
					domain:         "example.com",
					cookieName:     "abc123",
					containsDomain: true,
					matches:        true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			regexpMap := getDomainCookieNamesRegexp(tt.cookies)
			for _, subtest := range tt.subtests {
				re, ok := regexpMap[subtest.domain]
				require.Equal(t, subtest.containsDomain, ok, "domain match fail")
				if subtest.containsDomain {
					require.Equal(t, subtest.matches, re.MatchString(subtest.cookieName), "regexp fails")
				}
			}
		})
	}
}

func TestLootServiceSaveCookiesOneRequired(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

func TestLootServiceSaveCookiesOneRequiredWithoutDomain(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "www.example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "www.example.com",
			Required: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

func TestLootServiceSaveCookiesOneRequiredWithDot(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   ".example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

func TestLootServiceSaveCookiesTwoRequired(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		sort.Slice(sess.Cookies, func(i, j int) bool {
			if sess.Cookies[i].Name < sess.Cookies[j].Name {
				return true
			}
			return sess.Cookies[i].Domain < sess.Cookies[j].Domain
		})
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
				{
					Name:           "sessionid2",
					Value:          "Value2",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
		{
			Name:     "sessionid2",
			Domain:   "example.com",
			Required: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
		{
			Name:     "sessionid2",
			Value:    "Value2",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

func TestLootServiceSaveCookiesNoRequired(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		require.Fail(t, "sessionRepository should not be called")
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:   "sessionid",
			Domain: "example.com",
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)
}

func TestLootServiceSaveCookiesOneRequiredTwoCalls(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

func TestLootServiceSaveCookiesOneRequiredOneRegexp(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		sort.Slice(sess.Cookies, func(i, j int) bool {
			if sess.Cookies[i].Name < sess.Cookies[j].Name {
				return true
			}
			return sess.Cookies[i].Domain < sess.Cookies[j].Domain
		})
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
				{
					Name:           "sessionid2",
					Value:          "Value2",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   ".example.com",
			Required: true,
		},
		{
			Name:   ".*",
			Domain: ".example.com",
			Regexp: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
		{
			Name:     "sessionid2",
			Value:    "Value2",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

func TestLootServiceSaveCookiesOneRequiredSaveExpired(t *testing.T) {

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	websess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, websess)
	setLureURL(websess, "/abc/def")

	called := 0
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		called++
		require.Equal(t, &DBCapturedSession{
			LureURL:   "/abc/def",
			SessionID: websess.ID(),
			Cookies: []*SessionCookie{
				{
					Name:           "sessionid",
					Value:          "Value1",
					Path:           "/",
					Domain:         "example.com",
					ExpirationDate: 1643767322.123123,
					HTTPOnly:       true,
					Secure:         true,
					SameSite:       "no_restriction",
					Session:        false,
				},
			},
		}, sess)
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(1970, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	require.Equal(t, 1, called, "sessionRepository is called invalid number of times")
}

type mockSessionRepositoryFunc func(sess *DBCapturedSession) error

func (f mockSessionRepositoryFunc) SaveSession(sess *DBCapturedSession) error {
	return f(sess)
}

func TestLootServiceIsAuthenticatedWithoutSession(t *testing.T) {
	s := NewLootService(nil, nil, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	result := s.IsAuthenticated(c)
	require.False(t, result)
}

func TestLootServiceIsAuthenticatedWithSession(t *testing.T) {
	s := NewLootService(nil, nil, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	sess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, sess)

	result := s.IsAuthenticated(c)
	require.False(t, result)
}

func TestLootServiceIsAuthenticatedWithSessionAuthenticated(t *testing.T) {
	sessionRepo := mockSessionRepositoryFunc(func(sess *DBCapturedSession) error {
		return nil
	})
	s := NewLootService(nil, sessionRepo, []*SessionCookieConfig{
		{
			Name:     "sessionid",
			Domain:   "example.com",
			Required: true,
		},
	})

	app := fiber.New()
	c := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(c)
	store := session.New()
	sess, err := store.Get(c)
	require.NoError(t, err)
	setSession(c, sess)

	destURL := &url.URL{Scheme: "https", Host: "www.example.com", Path: "/"}
	err = s.SaveCookies(c, destURL, []*http.Cookie{
		{
			Name:     "sessionid",
			Value:    "Value1",
			Path:     "/",
			Domain:   "example.com",
			Expires:  time.Date(2022, 2, 2, 2, 2, 2, 123123000, time.UTC),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteNoneMode,
		},
	})
	require.NoError(t, err)

	result := s.IsAuthenticated(c)
	require.True(t, result)
}
