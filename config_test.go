package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/stretchr/testify/require"
)

func TestNewAppConfigDefaultValues(t *testing.T) {
	conf, err := newAppConfig(&configSource{
		provider: confmap.Provider(map[string]interface{}{
			"api_token":   "abc",
			"domain_name": "example.com",
			"tls_key":     "key.pem",
			"tls_cert":    "cert.pem",
		}, "."),
	})
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0:8080", conf.ListenAddr)
	require.Equal(t, "session_id", conf.SessionCookieName)
	require.Equal(t, 30*time.Minute, conf.SessionExpiration)
}

func TestNewAppConfigDotEnvFile(t *testing.T) {
	conf, err := newAppConfig(&configSource{
		provider: rawbytes.Provider([]byte(`
		API_TOKEN=abc
		LISTEN_ADDR=0.0.0.0:4041
		DOMAIN_NAME=example.com
		EXTERNAL_PORT=8091
		TLS_KEY=key.pem
		TLS_CERT=cert.pem
		SESSION.COOKIE_NAME=session_id2
		SESSION.EXPIRATION=1h
		`)),
		parser: &lowerCaseParser{dotenv.Parser()},
	})
	require.NoError(t, err)
	require.Equal(t, "abc", conf.APIToken)
	require.Equal(t, "0.0.0.0:4041", conf.ListenAddr)
	require.Equal(t, "example.com", conf.DomainName)
	require.Equal(t, "8091", conf.ExternalPort)
	require.Equal(t, "key.pem", conf.TLSKey)
	require.Equal(t, "cert.pem", conf.TLSCert)
	require.Equal(t, "session_id2", conf.SessionCookieName)
	require.Equal(t, 1*time.Hour, conf.SessionExpiration)
}

func TestNewAppConfigYamlFile(t *testing.T) {
	conf, err := newAppConfig(&configSource{
		provider: rawbytes.Provider([]byte(`
        api_token: abc
        listen_addr: 0.0.0.0:4041
        domain_name: example.com
        external_port: 8091
        tls_key: key.pem
        tls_cert: cert.pem
        session:
            cookie_name: session_id2
            expiration: 1h
        domain_mappings:
            - proxy: www.example.com
              target: www.google.com`)),
		parser: yaml.Parser(),
	})
	require.NoError(t, err)
	require.Equal(t, "abc", conf.APIToken)
	require.Equal(t, "0.0.0.0:4041", conf.ListenAddr)
	require.Equal(t, "example.com", conf.DomainName)
	require.Equal(t, "8091", conf.ExternalPort)
	require.Equal(t, "key.pem", conf.TLSKey)
	require.Equal(t, "cert.pem", conf.TLSCert)
	require.Equal(t, "session_id2", conf.SessionCookieName)
	require.Equal(t, 1*time.Hour, conf.SessionExpiration)
	require.Equal(t, []DomainMapping{
		{Proxy: "www.example.com", Target: "www.google.com"}}, conf.StaticDomainMappings)
}

func TestNewAppConfigDomainNameWithPort(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected string
	}{
		{
			name: "DomainWithoutPort",
			data: `
			API_TOKEN=abc
			LISTEN_ADDR=0.0.0.0:4041
			DOMAIN_NAME=example.com
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			SESSION.COOKIE_NAME=session_id2
			SESSION.EXPIRATION=1h
			`,
			expected: "example.com",
		},
		{
			name: "DomainWithPort",
			data: `
			API_TOKEN=abc
			LISTEN_ADDR=0.0.0.0:4041
			DOMAIN_NAME=example.com
			EXTERNAL_PORT=8091
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			SESSION.COOKIE_NAME=session_id2
			SESSION.EXPIRATION=1h
			`,
			expected: "example.com:8091",
		},
		{
			name: "DomainWithDefaultPort",
			data: `
			API_TOKEN=abc
			LISTEN_ADDR=0.0.0.0:4041
			DOMAIN_NAME=example.com
			EXTERNAL_PORT=443
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			SESSION.COOKIE_NAME=session_id2
			SESSION.EXPIRATION=1h
			`,
			expected: "example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := newAppConfig(&configSource{
				provider: rawbytes.Provider([]byte(tt.data)),
				parser:   &lowerCaseParser{dotenv.Parser()},
			})
			require.NoError(t, err)
			require.Equal(t, tt.expected, conf.DomainNameWithPort)
		})
	}
}

func TestNewAppConfigInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "InvalidExternalPort",
			data: `
            api_token: abc
            listen_addr: 0.0.0.0:4041
            domain_name: example.com
            external_port: abc
            tls_key: key.pem
            tls_cert: cert.pem
            session:
            cookie_name: session_id2
                expiration: 1h
            domain_mappings:
                - proxy: www.example.com
                  target: www.google.com`,
		},
		{
			name: "EmptyAPIToken",
			data: `
            listen_addr: 0.0.0.0:4041
            domain_name: example.com
            external_port: 8091
            tls_key: key.pem
            tls_cert: cert.pem
            session:
            cookie_name: session_id2
                expiration: 1h
            domain_mappings:
                - proxy: www.example.com
                  target: www.google.com`,
		},
	}

	for _, tt := range tests {
		_, err := newAppConfig(&configSource{
			provider: rawbytes.Provider([]byte(tt.data)),
			parser:   yaml.Parser(),
		})
		require.Error(t, err)
	}
}

func TestParseAppConfig(t *testing.T) {
	tests := []struct {
		name     string
		yamlFile string
		envFile  string
		testFS   fs.FS
	}{
		{
			name:     "YamlFileOnly",
			yamlFile: "config.yml",
			testFS: fstest.MapFS{
				"config.yml": &fstest.MapFile{
					Data: []byte(`
                    api_token: abc
                    tls_key: key.pem
                    tls_cert: cert.pem
                    listen_addr: 0.0.0.0:4041
                    domain_name: example.com`),
				},
			},
		},
		{
			name:    "DotEnvFileOnly",
			envFile: "config.env",
			testFS: fstest.MapFS{
				"config.env": &fstest.MapFile{
					Data: []byte(`
					API_TOKEN=abc
					TLS_KEY=key.pem
					TLS_CERT=cert.pem
					LISTEN_ADDR=0.0.0.0:4041
					DOMAIN_NAME=example.com`),
				},
			},
		},
		{
			name:     "YamlAndDotEnvFiles",
			yamlFile: "config.yml",
			envFile:  "config.env",
			testFS: fstest.MapFS{
				"config.yml": &fstest.MapFile{
					Data: []byte(`
                    api_token: abc
                    listen_addr: 0.0.0.0:4041
                    domain_name: example.com`),
				},
				"config.env": &fstest.MapFile{
					Data: []byte(`
					API_TOKEN=def
					TLS_KEY=key.pem
					TLS_CERT=cert.pem
					LISTEN_ADDR=0.0.0.0:4044
					DOMAIN_NAME=example1.com`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseAppConfig(tt.testFS, tt.yamlFile, tt.envFile)
			require.NoError(t, err)
		})
	}
}

func TestParseAppConfigDotEnvData(t *testing.T) {
	testFS := fstest.MapFS{
		"config.yml": &fstest.MapFile{
			Data: []byte(`
            api_token: abc
            listen_addr: 0.0.0.0:4041
            domain_name: example.com`),
		},
		"config.env": &fstest.MapFile{
			Data: []byte(`
			API_TOKEN=def
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			LISTEN_ADDR=0.0.0.0:4044
			DOMAIN_NAME=example1.com`),
		},
	}
	conf, err := parseAppConfig(testFS, "config.yml", "config.env")
	require.NoError(t, err)
	require.Equal(t, "def", conf.APIToken)
	require.Equal(t, "0.0.0.0:4044", conf.ListenAddr)
	require.Equal(t, "example1.com", conf.DomainName)
}