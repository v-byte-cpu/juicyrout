package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type RequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewProxyHandler(client RequestDoer,
	req RequestProcessor, resp ResponseProcessor) fiber.Handler {
	proxy := &proxyHandler{client, req, resp}
	return func(c *fiber.Ctx) error {
		return proxy.Handle(c)
	}
}

type proxyHandler struct {
	client RequestDoer
	req    RequestProcessor
	resp   ResponseProcessor
}

func (p *proxyHandler) Handle(c *fiber.Ctx) error {
	req := p.req.Process(c)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	p.resp.Process(c, resp)
	return nil
}
