package middleware

import (
	"strconv"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/log"

	echo "github.com/labstack/echo/v4"
)

type logContext struct {
	echo.Context
	logger *Logger
}

func withLogContext(ctx echo.Context, logger *Logger) *logContext {
	return &logContext{ctx, logger}
}

func (c *logContext) Logger() echo.Logger {
	return c.logger
}

//

func Log(l log.Logger, msg string) echo.MiddlewareFunc {
	logger := &Logger{Logger: l}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var (
				req   = c.Request()
				res   = c.Response()
				start = time.Now()
			)

			c = withLogContext(c, logger)

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			stop := time.Now()

			evt := l.Info().
				Str("request_id", res.Header().Get(echo.HeaderXRequestID)).
				Str("remote_ip", c.RealIP()).
				Str("host", req.Host).
				Str("method", req.Method).
				Str("uri", req.RequestURI).
				Str("user_agent", req.UserAgent()).
				Int("status", res.Status).
				Str("referer", req.Referer()).
				Dur("latency", stop.Sub(start)).
				Str("latency_human", stop.Sub(start).String())

			cl := req.Header.Get(echo.HeaderContentLength)
			if cl == "" {
				cl = "0"
			}
			evt.
				Str("bytes_in", cl).
				Str("bytes_out", strconv.FormatInt(res.Size, 10))

			if err != nil {
				evt.Err(err)
			}

			evt.Msg(msg)

			return err
		}
	}
}
