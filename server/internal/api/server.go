// Package api wires the HTTP surface of the platform: public storefront
// endpoints, B2B portal, admin panel, chatbot, webhooks and CRM.
package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"menguz/internal/badger"
	"menguz/internal/config"
	"menguz/internal/middleware"
	"menguz/internal/sqlite"
)

type Server struct {
	Cfg    *config.Config
	Store  *badger.Store
	Ana    *sqlite.DB
	Router *echo.Echo

	Limiter     *middleware.RateLimiter
	CheckoutLim *middleware.RateLimiter
	ChatLim     *middleware.RateLimiter
}

func NewServer(cfg *config.Config, st *badger.Store, ana *sqlite.DB) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	s := &Server{
		Cfg:         cfg,
		Store:       st,
		Ana:         ana,
		Router:      e,
		Limiter:     middleware.NewRateLimiter(100, 100),
		CheckoutLim: middleware.NewRateLimiter(20, 10),
		ChatLim:     middleware.NewRateLimiter(30, 15),
	}

	e.Use(middleware.SecurityHeaders)
	e.Use(middleware.CORS(cfg.CORSOrigins))
	e.Use(s.Limiter.Limit())

	s.routes()
	return s
}

func (s *Server) Start(addr string) error {
	return s.Router.Start(addr)
}

func (s *Server) ShutdownGracefully() error {
	// Echo graceful stop
	ctx, cancel := stdContextWithTimeout(10 * time.Second)
	defer cancel()
	return s.Router.Shutdown(ctx)
}

func newID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

func nowUTC() time.Time { return time.Now().UTC() }

// badRequest / notFound helpers keep handlers terse.
func badRequest(msg string) *echo.HTTPError { return echo.NewHTTPError(http.StatusBadRequest, msg) }
