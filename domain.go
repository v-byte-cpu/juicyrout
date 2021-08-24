package main

import (
	"net/url"
	"regexp"
	"strings"
)

type DomainConverter interface {
	// ToProxyDomain converts target original domain to proxy domain
	ToProxyDomain(domain string) string
	// ToTargetDomain converts proxy domain to target original domain
	ToTargetDomain(domain string) string
	// ToTargetURL converts proxy URL to target original URL
	ToTargetURL(proxyURL string) string
	AddStaticMapping(proxyDomain string, targetDomain string)
}

func NewDomainConverter(baseDomain string) DomainConverter {
	baseDomain = "." + baseDomain
	return &domainConverter{
		baseDomain: baseDomain,
		// remove port from cookie domain
		cookieBaseDomain: strings.Split(baseDomain, ":")[0],
		toTargetMap:      make(map[string]string),
		toProxyMap:       make(map[string]string),
		toTargetRegex:    regexp.MustCompile(`(--)|(\w\-\w)`),
	}
}

type domainConverter struct {
	baseDomain       string
	cookieBaseDomain string
	toTargetMap      map[string]string
	toProxyMap       map[string]string

	toTargetRegex *regexp.Regexp
}

func (c *domainConverter) ToProxyDomain(domain string) string {
	if v, ok := c.toProxyMap[domain]; ok {
		return v
	}
	return c.toProxy(domain, c.baseDomain)
}

func (*domainConverter) toProxy(domain, baseDomain string) string {
	if strings.HasSuffix(domain, baseDomain) {
		return domain
	}
	var sb strings.Builder
	for _, ch := range domain {
		switch ch {
		case '-':
			sb.WriteString("--")
		case '.':
			sb.WriteByte('-')
		default:
			sb.WriteRune(ch)
		}
	}
	sb.WriteString(baseDomain)
	return sb.String()
}

func (c *domainConverter) ToTargetDomain(domain string) string {
	if v, ok := c.toTargetMap[domain]; ok {
		return v
	}
	domain = strings.TrimSuffix(domain, c.baseDomain)

	idx := c.toTargetRegex.FindAllStringIndex(domain, -1)
	var sb strings.Builder
	start := 0
	for _, loc := range idx {
		sb.WriteString(domain[start:loc[0]])
		if loc[1]-loc[0] == 2 {
			sb.WriteByte('-')
		} else {
			sb.WriteByte(domain[loc[0]])
			sb.WriteByte('.')
			sb.WriteByte(domain[loc[1]-1])
		}
		start = loc[1]
	}
	sb.WriteString(domain[start:])
	return sb.String()
}

func (c *domainConverter) ToTargetURL(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return ""
	}
	if len(u.Host) > 0 {
		u.Host = c.ToTargetDomain(u.Host)
	}
	return u.String()
}

func (c *domainConverter) AddStaticMapping(proxyDomain string, targetDomain string) {
	c.toTargetMap[proxyDomain] = targetDomain
	c.toProxyMap[targetDomain] = proxyDomain
}
