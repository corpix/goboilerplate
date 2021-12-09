package server

import (
	echo "github.com/labstack/echo/v4"

	serverErrors "git.backbone/corpix/goboilerplate/pkg/server/errors"
)

type (
	HTTPError = echo.HTTPError
	Error     = serverErrors.Error
)

func DefaultHTTPErrorHandler(err error, c Context) {
	if _, ok := err.(*HTTPError); ok {
		c.Echo().DefaultHTTPErrorHandler(err, c)
		return
	}

	//

	code := StatusInternalServerError
	r := NewErrorResult(code, "")

	if e, ok := err.(*Error); ok {
		r.Error.Code = e.Code
		r.Error.Message = e.Error()
	}

	_ = c.JSON(r.Error.Code, r)
}

var NewError = serverErrors.NewError
