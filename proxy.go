package main

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
)

type RequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewProxyHandler(log *zerolog.Logger, client RequestDoer,
	req RequestProcessor, resp ResponseProcessor) fiber.Handler {
	proxy := &proxyHandler{log, client, req, resp}
	return func(c *fiber.Ctx) error {
		return proxy.Handle(c)
	}
}

type proxyHandler struct {
	log    *zerolog.Logger
	client RequestDoer
	req    RequestProcessor
	resp   ResponseProcessor
}

func (p *proxyHandler) Handle(c *fiber.Ctx) error {
	var req *http.Request
	defer func() {
		if err := recover(); err != nil {
			p.log.Error().Any("request", req).Msg("panic in proxy for request")
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
