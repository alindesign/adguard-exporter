package http

import (
	"context"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/alindesign/adguard-exporter/internal/config"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
)

type Http struct {
	e    *echo.Echo
	addr string
}

func NewHttp(configuration *config.Config) *Http {
	addr := net.JoinHostPort(configuration.Host, configuration.Port)

	e := echo.New()
	e.HideBanner = true

	e.GET("/metrics", echoprometheus.NewHandler())

	if configuration.Debug {
		e.GET("/debug/*", echo.WrapHandler(http.DefaultServeMux))
	}

	return &Http{e, addr}
}

func (h *Http) Serve() error {
	log.Printf("Starting http server on %s", h.addr)
	return h.e.Start(h.addr)
}

func (h *Http) Stop(ctx context.Context) error {
	return h.e.Shutdown(ctx)
}
