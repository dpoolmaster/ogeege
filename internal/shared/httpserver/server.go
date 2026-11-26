// Package httpserver собирает HTTP-маршруты и middleware приложения.
package httpserver

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

// Options содержит зависимости HTTP-сервера.
type Options struct {
	Env              string
	Logger           *slog.Logger
	MaxBodyBytes     int64
	Pinger           Pinger
	MigrationChecker ReadinessChecker
	CSRF             gin.HandlerFunc
	Routes           func(*gin.Engine)
}

// NewRouter создаёт Gin-роутер с обязательным набором middleware.
func NewRouter(opts Options) *gin.Engine {
	router := gin.New()
	router.Use(
		RequestID(),
		Recovery(opts.Logger),
		RequestLogger(opts.Logger),
		SecurityHeaders(opts.Env),
		BodyLimit(opts.MaxBodyBytes),
	)
	if opts.CSRF != nil {
		router.Use(opts.CSRF)
	}

	router.GET("/healthz", HealthHandler())
	router.GET("/readyz", ReadyHandler(opts.Pinger, opts.MigrationChecker))

	if opts.Routes != nil {
		opts.Routes(router)
	}

	return router
}
