package main

import (
	"regexp"
	"strings"
)

type DomainConverter interface {
	// ToProxy converts target original domain to proxy domain
	ToProxy(domain string) string
	// ToProxyCoolie converts cookie target original domain to proxy domain
	ToProxyCookie(domain string) string
	// ToTarget converts proxy domain to target original domain
	ToTarget(domain string) string
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
		toProxyCookieMap: make(map[string]string),
		toTargetRegex:    regexp.MustCompile(`(--)|(\w\-\w)`),
	}
}

type domainConverter struct {
	baseDomain       string
	cookieBaseDomain string
	toTargetMap      map[string]string
	toProxyMap       map[string]string
	toProxyCookieMap map[string]string

	toTargetRegex *regexp.Regexp
}

func (c *domainConverter) ToProxy(domain string) string {
	if v, ok := c.toProxyMap[domain]; ok {
		return v
	}
	return c.toProxy(domain, c.baseDomain)
}

func (c *domainConverter) ToProxyCookie(domain string) string {
	if domain == "" {
		return ""
	}
	if v, ok := c.toProxyCookieMap[domain]; ok {
		return v
	}
	domain = strings.TrimPrefix(domain, ".")
	domain = c.toProxy(domain, c.cookieBaseDomain)
	return domain
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

func (c *domainConverter) ToTarget(domain string) string {
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

func (c *domainConverter) AddStaticMapping(proxyDomain string, targetDomain string) {
	c.toTargetMap[proxyDomain] = targetDomain
	c.toProxyMap[targetDomain] = proxyDomain
	c.toProxyCookieMap[targetDomain] = strings.Split(proxyDomain, ":")[0]
}
