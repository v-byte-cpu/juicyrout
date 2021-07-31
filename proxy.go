package main

import (
	"log"
	"net/http"
)

type RequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type proxyHandler struct {
	client RequestDoer
	req    RequestProcessor
	resp   ResponseProcessor
}

func NewProxyHandler(client RequestDoer,
	req RequestProcessor, resp ResponseProcessor) http.Handler {
	return &proxyHandler{client, req, resp}
}

func (p *proxyHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	req = p.req.Process(req)
	resp, err := p.client.Do(req)
	if err != nil {
		log.Println("error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	p.resp.Process(w, resp)
}
