package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ResponseProcessor interface {
	Process(c *fiber.Ctx, resp *http.Response)
}

func NewResponseProcessor(conv DomainConverter, urlProc, htmlProc RegexProcessor) ResponseProcessor {
	return &responseProcessor{conv, urlProc, htmlProc}
}

type responseProcessor struct {
	conv     DomainConverter
	urlProc  RegexProcessor
	htmlProc RegexProcessor
}

func (p *responseProcessor) Process(c *fiber.Ctx, resp *http.Response) {
	p.convertCORS(resp)
	p.removePolicyHeaders(resp)
	p.convertLocation(resp)
	p.writeCookies(c, resp)
	p.writeHeaders(c, resp)
	p.writeBody(c, resp)
	c.Status(resp.StatusCode)
}

func (p *responseProcessor) convertCORS(resp *http.Response) {
	originHeader := resp.Request.Header["Origin"]
	if len(originHeader) > 0 {
		proxyOrigin := p.conv.ToProxyDomain(originHeader[0])
		resp.Header["Access-Control-Allow-Origin"] = []string{proxyOrigin}
		resp.Header["Access-Control-Allow-Credentials"] = []string{"true"}
	}
}

func (*responseProcessor) removePolicyHeaders(resp *http.Response) {
	resp.Header.Del("Content-Security-Policy")
	resp.Header.Del("Content-Security-Policy-Report-Only")
	resp.Header.Del("Cross-Origin-Opener-Policy")
	resp.Header.Del("Cross-Origin-Opener-Policy-Report-Only")
	resp.Header.Del("Cross-Origin-Embedder-Policy")
	resp.Header.Del("Cross-Origin-Embedder-Policy-Report-Only")
	resp.Header.Del("Report-To")
}

func (p *responseProcessor) convertLocation(resp *http.Response) {
	p.convertHeaderDomain(resp, "Location")
	p.convertHeaderDomain(resp, "Content-Location")
}

func (p *responseProcessor) convertHeaderDomain(resp *http.Response, headerName string) {
	value := resp.Header[headerName]
	if len(value) == 0 {
		return
	}
	u, err := url.Parse(value[0])
	if err != nil {
		return
	}
	if len(u.Host) > 0 {
		u.Host = p.conv.ToProxyDomain(u.Host)
	}
	resp.Header[headerName] = []string{u.String()}
}

func (*responseProcessor) writeCookies(c *fiber.Ctx, resp *http.Response) {
	if resp.Request.Method == http.MethodOptions {
		return
	}
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		log.Println(resp.Request.URL, "set cookie:", cookie.String())
	}
	cookieJar := c.Locals("cookieJar").(http.CookieJar)
	cookieJar.SetCookies(resp.Request.URL, cookies)

	resp.Header.Del("Set-Cookie")
}

func (*responseProcessor) writeHeaders(c *fiber.Ctx, resp *http.Response) {
	for header, values := range resp.Header {
		for _, v := range values {
			c.Response().Header.Add(header, v)
		}
	}
}

//nolint:errcheck
func (p *responseProcessor) writeBody(w io.Writer, resp *http.Response) {
	contentType := p.getContentType(resp)
	if strings.Contains(contentType, "text/html") {
		p.htmlProc.ProcessAll(w, resp.Body)
		return
	}
	if processContentTypeRegexp.MatchString(contentType) {
		p.urlProc.ProcessAll(w, resp.Body)
	} else {
		io.Copy(w, resp.Body)
	}
}

func (*responseProcessor) getContentType(resp *http.Response) string {
	contentType := resp.Header["Content-Type"]
	if len(contentType) > 0 {
		return strings.ToLower(contentType[0])
	}
	return ""
}
