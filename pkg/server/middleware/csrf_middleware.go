package middleware

import (
	"net/http"
	"strings"

	echo "github.com/labstack/echo/v4"

	"git.backbone/corpix/goboilerplate/pkg/server/csrf"
)

const (
	CSRFContextKey    = csrf.ContextKey
	CSRFParameterName = csrf.ParameterName
)

func NewCSRF(t *csrf.CSRF, methods []string) echo.MiddlewareFunc {
	if methods == nil {
		methods = []string{
			http.MethodPost,
			http.MethodPut,
		}
	}
	methodIndex := make(map[string]struct{}, len(methods))

	for _, method := range methods {
		methodIndex[strings.ToUpper(method)] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			_, signatureRequired := methodIndex[c.Request().Method]
			if signatureRequired {
				err := t.ValidateContext(c)
				if err != nil {
					return err
				}
			}

			c.Set(CSRFContextKey, t)

			return next(c)
		}
	}
}
