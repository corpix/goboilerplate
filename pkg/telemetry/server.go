package telemetry

import (
	"context"
	"net"
	"net/http"

	"git.backbone/corpix/goboilerplate/pkg/log"
	"git.backbone/corpix/goboilerplate/pkg/middleware"

	echo "github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Listener net.Listener

type Server struct {
	config   Config
	log      log.Logger
	listener net.Listener
	srv      *echo.Echo
	handler  http.Handler
}

func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:         s.config.Addr,
		ReadTimeout:  s.config.Timeout.Read,
		WriteTimeout: s.config.Timeout.Write,
	}

	s.srv.Listener = s.listener

	err := s.srv.StartServer(srv)
	if err == http.ErrServerClosed {
		s.log.
			Warn().
			Str("addr", s.config.Addr).
			Msg("server shutdown")
		return nil
	}

	return err
}

func (s *Server) Handle(ctx echo.Context) error {
	s.handler.ServeHTTP(
		ctx.Response().Writer,
		ctx.Request(),
	)
	return nil
}

func (s *Server) Close() error {
	err := s.srv.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// see: https://github.com/swaggo/swag#declarative-comments-format
// @title Telemetry
// @version 1.0
// @description Prometheus telemetry endpoint

// @BasePath /
// @Router / [get]
// @Produce text/plain
// @Summary Respond with prometheus metrics
// @Success 200
func New(c Config, l log.Logger, r *Registry, lr Listener) *Server {
	var addr string

	if lr != nil {
		addr = lr.Addr().String()
	} else {
		addr = c.Addr
	}

	l = l.With().Str("component", Subsystem).Str("listener", addr).Logger()

	h := promhttp.InstrumentMetricHandler(
		r,
		promhttp.HandlerFor(
			r,
			promhttp.HandlerOpts{ErrorLog: log.Std(l)},
		),
	)

	e := echo.New()
	e.HideBanner = true
	e.Logger = &middleware.Logger{Logger: l}

	e.Use(middleware.Log(l, "processed telemetry request"))
	e.Use(echomw.BodyLimit("0"))
	e.Use(echomw.Recover())

	middleware.MountSwagger(e, "/swagger")

	s := &Server{
		config:   c,
		log:      l,
		listener: lr,
		srv:      e,
		handler:  h,
	}

	e.GET(c.Path, s.Handle)

	return s
}
