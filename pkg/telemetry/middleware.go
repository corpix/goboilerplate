package telemetry

import (
	"net/http"
	"strconv"
	"time"

	echo "github.com/labstack/echo/v4"
)

func approxRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}

func Middleware(r *Registry, subsystem string) echo.MiddlewareFunc {
	reqTot := NewCounterVec(
		CounterOpts{
			Name: Name(subsystem, "requests", "total"),
			Help: "How many HTTP requests processed, partitioned by status code and HTTP method.",
		},
		[]string{"code", "method", "path"},
	)

	reqDur := NewHistogramVec(
		HistogramOpts{
			Name: Name(subsystem, "request", "duration", "seconds"),
			Help: "The HTTP request latencies in seconds.",
		},
		[]string{"code", "method", "path"},
	)

	reqSz := NewHistogramVec(
		HistogramOpts{
			Name: Name(subsystem, "request", "size", "bytes"),
			Help: "The HTTP request sizes in bytes.",
		},
		[]string{"code", "method", "path"},
	)

	resSz := NewHistogramVec(
		HistogramOpts{
			Name: Name(subsystem, "response", "size", "bytes"),
			Help: "The HTTP response sizes in bytes.",
		},
		[]string{"code", "method", "path"},
	)

	//

	r.MustRegister(reqTot)
	r.MustRegister(reqDur)
	r.MustRegister(reqSz)
	r.MustRegister(resSz)

	//

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()
			reqSize := approxRequestSize(req)

			//

			err := next(c)
			if err != nil {
				// continue on error to count metrics
				c.Error(err)
			}

			//

			status := strconv.Itoa(c.Response().Status)
			path := req.URL.Path
			elapsed := float64(time.Since(start)) / float64(time.Second)
			resSize := float64(c.Response().Size)

			//

			reqTot.WithLabelValues(status, c.Request().Method, path).Inc()
			reqDur.WithLabelValues(status, c.Request().Method, path).Observe(elapsed)
			reqSz.WithLabelValues(status, c.Request().Method, path).Observe(float64(reqSize))
			resSz.WithLabelValues(status, c.Request().Method, path).Observe(resSize)

			return err
		}
	}
}
