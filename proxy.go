package main

import (
	"log"
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
	var req *http.Request
	defer func() {
		if err := recover(); err != nil {
			log.Println("panic in proxy for request: ", req)
			panic(err)
		}
	}()
	req = p.req.Process(c)
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	p.resp.Process(c, resp)
	return nil
}
