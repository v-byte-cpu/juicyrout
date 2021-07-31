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
	return &domainConverter{
		baseDomain:    "." + baseDomain,
		toTargetMap:   make(map[string]string),
		toProxyMap:    make(map[string]string),
		toTargetRegex: regexp.MustCompile("(--)|(\\w\\-\\w)"),
	}
}

type domainConverter struct {
	baseDomain  string
	toTargetMap map[string]string
	toProxyMap  map[string]string

	toTargetRegex *regexp.Regexp
}

func (c *domainConverter) ToProxy(domain string) string {
	if v, ok := c.toProxyMap[domain]; ok {
		return v
	}
	if strings.HasSuffix(domain, c.baseDomain) {
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
	sb.WriteString(c.baseDomain)
	return sb.String()
}

func (c *domainConverter) ToProxyCookie(domain string) string {
	domain = strings.TrimPrefix(domain, ".")
	domain = c.ToProxy(domain)
	return domain
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
}
