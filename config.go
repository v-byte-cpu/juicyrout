package main

import (
	"fmt"
	"io/fs"
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
)

type appConfig struct {
	APIToken             string          `koanf:"api_token" validate:"required"`
	ListenAddr           string          `koanf:"listen_addr" validate:"required"`
	DomainName           string          `koanf:"domain_name" validate:"required"`
	ExternalPort         string          `koanf:"external_port" validate:"omitempty,numeric"`
	TLSKey               string          `koanf:"tls_key" validate:"required"`
	TLSCert              string          `koanf:"tls_cert" validate:"required"`
	SessionCookieName    string          `koanf:"session.cookie_name" validate:"required"`
	SessionExpiration    time.Duration   `koanf:"session.expiration" validate:"required"`
	StaticDomainMappings []DomainMapping `koanf:"domain_mappings"`
	DomainNameWithPort   string
}

type DomainMapping struct {
	Proxy  string `koanf:"proxy"`
	Target string `koanf:"target"`
}

type configSource struct {
	provider koanf.Provider
	parser   koanf.Parser
}

func parseAppConfig(fsys fs.FS, yamlFile string, envFile string) (*appConfig, error) {
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
		"listen_addr":         "0.0.0.0:8080",
		"session.cookie_name": "session_id",
		"session.expiration":  30 * time.Minute,
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
	validate := validator.New()
	err = validate.Struct(&conf)
	if conf.ExternalPort == "" || conf.ExternalPort == "443" {
		conf.DomainNameWithPort = conf.DomainName
	} else {
		conf.DomainNameWithPort = fmt.Sprintf("%s:%s", conf.DomainName, conf.ExternalPort)
	}
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
