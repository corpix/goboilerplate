package response

import (
	echo "github.com/labstack/echo/v4"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

const FinalizerContextKey = "response-finalizer"

type (
	Context                 = echo.Context
	Finalizer               interface{}
	FinalizerDispatchOption = func(Context, Finalizer) (bool, error)
)

//

func SetFinalizer(ctx Context, f Finalizer) {
	ctx.Set(FinalizerContextKey, f)
}

func GetFinalizer(ctx Context) Finalizer {
	return ctx.Get(FinalizerContextKey)
}

//

func DispatchFinalizer(ctx Context, r Finalizer, options ...FinalizerDispatchOption) error {
	var (
		match bool
		err   error
	)

	for _, op := range options {
		match, err = op(ctx, r)
		if err != nil {
			return err
		}
		if match {
			return nil
		}
	}

	switch rr := r.(type) {
	case *RedirectFinalizer:
		return ctx.Redirect(rr.Code, rr.URL)
	case *NoContentFinalizer:
		return ctx.NoContent(rr.Code)
	case *JSONFinalizer:
		return ctx.JSON(rr.Code, rr.JSON)
	case *HTMLFinalizer:
		return ctx.HTML(rr.Code, rr.HTML)
	case *HTMLBlobFinalizer:
		return ctx.HTMLBlob(rr.Code, rr.HTML)
	case *StringFinalizer:
		return ctx.String(rr.Code, rr.String)
	default:
		return errors.Errorf("failed to dispatch response type %T, no match", r)
	}
}
