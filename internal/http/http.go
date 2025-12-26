package http

import (
	"context"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/alindesign/adguard-exporter/internal/config"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
)

type Http struct {
	e *echo.Echo

	ready    bool
	healthy  bool
	healthMu *sync.Mutex
	addr     string
}

func NewHttp(configuration *config.Config) *Http {
	e := echo.New()
	e.HideBanner = true

	e.GET("/metrics", echoprometheus.NewHandler())

	if configuration.Debug {
		e.GET("/debug/*", echo.WrapHandler(http.DefaultServeMux))
	}

	instance := &Http{
		e:        e,
		ready:    false,
		healthMu: &sync.Mutex{},
		addr:     net.JoinHostPort(configuration.Host, configuration.Port),
	}

	instance.e.GET("/healthz", instance.healthz())
	instance.e.GET("/readyz", instance.readyz())

	return instance
}

func (h *Http) Serve() error {
	log.Printf("Starting http server on %s", h.addr)
	return h.e.Start(h.addr)
}

func (h *Http) Stop(ctx context.Context) error {
	return h.e.Shutdown(ctx)
}

func (h *Http) Ready(state bool) {
	h.healthMu.Lock()
	defer h.healthMu.Unlock()
	h.ready = state
}

func (h *Http) Healthy(state bool) {
	h.healthMu.Lock()
	defer h.healthMu.Unlock()
	h.healthy = state
}

func (h *Http) healthz() echo.HandlerFunc {
	return func(c echo.Context) error {
		h.healthMu.Lock()
		defer h.healthMu.Unlock()
		code := http.StatusOK
		if !h.healthy {
			code = http.StatusServiceUnavailable
		}
		return c.NoContent(code)
	}
}

func (h *Http) readyz() echo.HandlerFunc {
	return func(c echo.Context) error {
		h.healthMu.Lock()
		defer h.healthMu.Unlock()
		code := http.StatusOK
		if !h.ready {
			code = http.StatusServiceUnavailable
		}
		return c.NoContent(code)
	}
}
