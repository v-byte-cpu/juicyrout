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
			"api_token":     "abc",
			"domain_name":   "example.com",
			"tls_key":       "key.pem",
			"tls_cert":      "cert.pem",
			"phishlet_file": "phishlet.yml",
		}, "."),
	})
	require.NoError(t, err)
	require.Equal(t, "api", conf.APISubdomain)
	require.Equal(t, "0.0.0.0:8080", conf.ListenAddr)
	require.Equal(t, "session_id", conf.SessionCookieName)
	require.Equal(t, 30*time.Minute, conf.SessionExpiration)
	require.Equal(t, "file", conf.DBType)
	require.Equal(t, "creds.jsonl", conf.CredsFile)
	require.Equal(t, "lures.yaml", conf.LuresFile)
}

func TestNewAppConfigDotEnvFile(t *testing.T) {
	conf, err := newAppConfig(&configSource{
		provider: rawbytes.Provider([]byte(`
		API_TOKEN=abc
		API_SUBDOMAIN=appi1
		LISTEN_ADDR=0.0.0.0:4041
		DOMAIN_NAME=example.com
		EXTERNAL_PORT=8091
		TLS_KEY=key.pem
		TLS_CERT=cert.pem
		PHISHLET_FILE=phishlet.yml
		SESSION.COOKIE_NAME=session_id2
		SESSION.EXPIRATION=1h
		`)),
		parser: &lowerCaseParser{dotenv.Parser()},
	})
	require.NoError(t, err)
	require.Equal(t, "abc", conf.APIToken)
	require.Equal(t, "appi1", conf.APISubdomain)
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
        api_subdomain: appi1
        listen_addr: 0.0.0.0:4041
        domain_name: example.com
        external_port: 8091
        tls_key: key.pem
        tls_cert: cert.pem
        phishlet_file: phishlet.yml
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
	require.Equal(t, "appi1", conf.APISubdomain)
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
		name           string
		data           string
		expectedDomain string
		expectedAPI    string
	}{
		{
			name: "DomainWithoutPort",
			data: `
			API_TOKEN=abc
			LISTEN_ADDR=0.0.0.0:4041
			DOMAIN_NAME=example.com
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			PHISHLET_FILE=phishlet.yml
			SESSION.COOKIE_NAME=session_id2
			SESSION.EXPIRATION=1h
			`,
			expectedDomain: "example.com",
			expectedAPI:    "api.example.com",
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
			PHISHLET_FILE=phishlet.yml
			SESSION.COOKIE_NAME=session_id2
			SESSION.EXPIRATION=1h
			`,
			expectedDomain: "example.com:8091",
			expectedAPI:    "api.example.com:8091",
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
			PHISHLET_FILE=phishlet.yml
			SESSION.COOKIE_NAME=session_id2
			SESSION.EXPIRATION=1h
			`,
			expectedDomain: "example.com",
			expectedAPI:    "api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf, err := newAppConfig(&configSource{
				provider: rawbytes.Provider([]byte(tt.data)),
				parser:   &lowerCaseParser{dotenv.Parser()},
			})
			require.NoError(t, err)
			require.Equal(t, tt.expectedDomain, conf.DomainNameWithPort)
			require.Equal(t, tt.expectedAPI, conf.APIHostname)
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
            phishlet_file: phishlet.yml
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
            phishlet_file: phishlet.yml
            session:
                cookie_name: session_id2
                expiration: 1h
            domain_mappings:
                - proxy: www.example.com
                  target: www.google.com`,
		},
		{
			name: "InvalidDBType",
			data: `
            api_token: abc
            listen_addr: 0.0.0.0:4041
            domain_name: example.com
            external_port: 8091
            tls_key: key.pem
            tls_cert: cert.pem
            phishlet_file: phishlet.yml
            db_type: non_existing_type
            session:
                cookie_name: session_id2
                expiration: 1h
            domain_mappings:
                - proxy: www.example.com
                  target: www.google.com`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newAppConfig(&configSource{
				provider: rawbytes.Provider([]byte(tt.data)),
				parser:   yaml.Parser(),
			})
			require.Error(t, err)
		})
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
                    phishlet_file: phishlet.yml
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
					PHISHLET_FILE=phishlet.yml
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
					PHISHLET_FILE=phishlet.yml
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

func TestParseAppConfigDotEnvDataOverride(t *testing.T) {
	testFS := fstest.MapFS{
		"config.yml": &fstest.MapFile{
			Data: []byte(`
            api_token: abc
            phishlet_file: phishlet0.yml
            listen_addr: 0.0.0.0:4041
            domain_name: example.com`),
		},
		"config.env": &fstest.MapFile{
			Data: []byte(`
			API_TOKEN=def
			PHISHLET_FILE=phishlet.yml
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			LISTEN_ADDR=0.0.0.0:4044
			DOMAIN_NAME=example1.com`),
		},
	}
	conf, err := parseAppConfig(testFS, "config.yml", "config.env")
	require.NoError(t, err)
	require.Equal(t, "def", conf.APIToken)
	require.Equal(t, "phishlet.yml", conf.PhishletFile)
	require.Equal(t, "0.0.0.0:4044", conf.ListenAddr)
	require.Equal(t, "example1.com", conf.DomainName)
}

func TestParsePhishletConfig(t *testing.T) {
	testFS := fstest.MapFS{
		"phishlet.yml": &fstest.MapFile{
			Data: []byte(`
            invalid_auth_url: https://www.example.com/
            login_url: https://www.example.com/accounts/login
            js_files:
                - script.js
                - data/script2.js
            `),
		},
		"script.js": &fstest.MapFile{
			Data: []byte(`console.log("Hello!")`),
		},
		"data/script2.js": &fstest.MapFile{
			Data: []byte(`console.log("Hello2!")`),
		},
	}
	conf, err := parsePhishletConfig(testFS, "phishlet.yml")
	require.NoError(t, err)
	require.Equal(t, "https://www.example.com/", conf.InvalidAuthURL)
	require.Equal(t, "https://www.example.com/accounts/login", conf.LoginURL)
	require.Equal(t, []string{"script.js", "data/script2.js"}, conf.JsFiles)
	require.Equal(t, []string{`console.log("Hello!")`, `console.log("Hello2!")`}, conf.JsFilesBody)
}

func TestParsePhishletConfigInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{
			name: "EmptyInvalidAuthURL",
			data: `
            login_url: https://www.example.com/accounts/login
            js_files:
            - script.js`,
		},
		{
			name: "InvalidLoginURL",
			data: `
            invalid_auth_url: https://www.example.com/
            login_url: 123abc
            js_files:
            - script.js`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFS := fstest.MapFS{
				"phishlet.yml": &fstest.MapFile{
					Data: []byte(tt.data),
				},
				"script.js": &fstest.MapFile{},
			}
			_, err := parsePhishletConfig(testFS, "phishlet.yml")
			require.Error(t, err)
		})
	}
}

func TestSetupAppConfig(t *testing.T) {
	testFS := fstest.MapFS{
		"config.yml": &fstest.MapFile{
			Data: []byte(`
            api_token: abc
            phishlet_file: phishlet0.yml
            listen_addr: 0.0.0.0:4041
            domain_name: example.com`),
		},
		"config.env": &fstest.MapFile{
			Data: []byte(`
			API_TOKEN=def
			PHISHLET_FILE=configs/phishlet.yml
			TLS_KEY=key.pem
			TLS_CERT=cert.pem
			LISTEN_ADDR=0.0.0.0:4044
			DOMAIN_NAME=example1.com`),
		},
		"configs/phishlet.yml": &fstest.MapFile{
			Data: []byte(`
            invalid_auth_url: https://www.example.com/
            login_url: https://www.example.com/accounts/login
            js_files:
                - script.js
                - data/script2.js
            `),
		},
		"configs/script.js": &fstest.MapFile{
			Data: []byte(`console.log("Hello!")`),
		},
		"configs/data/script2.js": &fstest.MapFile{
			Data: []byte(`console.log("Hello2!")`),
		},
	}
	conf, err := setupAppConfig(testFS, "config.yml", "config.env")
	require.NoError(t, err)
	require.Equal(t, "def", conf.APIToken)
	require.Equal(t, "configs/phishlet.yml", conf.PhishletFile)
	require.Equal(t, "0.0.0.0:4044", conf.ListenAddr)
	require.Equal(t, "key.pem", conf.TLSKey)
	require.Equal(t, "cert.pem", conf.TLSCert)
	require.Equal(t, "example1.com", conf.DomainName)
	require.Equal(t, "https://www.example.com/", conf.Phishlet.InvalidAuthURL)
	require.Equal(t, "https://www.example.com/accounts/login", conf.Phishlet.LoginURL)
	require.Equal(t, []string{"script.js", "data/script2.js"}, conf.Phishlet.JsFiles)
	require.Equal(t, []string{`console.log("Hello!")`, `console.log("Hello2!")`}, conf.Phishlet.JsFilesBody)
}
