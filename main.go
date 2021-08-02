package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// TODO cli args
func main() {
	var port string
	flag.StringVar(&port, "p", "8091", "listening port")
	flag.Parse()
	// TODO static map www.example.com -> mail.com
	// TODO configure transport
	client := &http.Client{}
	conv := NewDomainConverter("host.juicyrout:" + port)
	req := NewRequestProcessor(conv)
	resp := NewResponseProcessor(conv)

	app := fiber.New(fiber.Config{
		StreamRequestBody:     true,
		DisableStartupMessage: true,
		IdleTimeout:           10 * time.Second,
	})
	app.Use(recover.New())
	app.All("/*", NewProxyHandler(client, req, resp))
	if err := app.ListenTLS(":"+port, "cert.pem", "key.pem"); err != nil {
		log.Println("listen error", err)
	}
}
