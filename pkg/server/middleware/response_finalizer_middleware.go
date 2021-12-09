package middleware

import (
	echo "github.com/labstack/echo/v4"

	"git.backbone/corpix/goboilerplate/pkg/server/response"
)

const ResponseFinalizerContextKey = response.FinalizerContextKey

func NewResponseFinalizer(options ...response.FinalizerDispatchOption) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			if err != nil {
				return err
			}

			finalizer := response.GetFinalizer(c)
			if finalizer == nil {
				return nil
			}

			return response.DispatchFinalizer(c, finalizer, options...)
		}
	}
}
