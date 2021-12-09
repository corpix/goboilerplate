package session

import (
	echo "github.com/labstack/echo/v4"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

type Store interface {
	Load() (error, bool)
	Save() error
	Drop() error
	Session() *Session
}

//

func GetStore(c echo.Context) (Store, bool) {
	v := c.Get(StoreContextKey)
	if v == nil {
		return nil, false
	}

	s, ok := v.(Store)

	return s, ok
}

func MustGetStore(c echo.Context) Store {
	s, ok := GetStore(c)
	if !ok {
		panic(errors.Errorf(
			"failed to load session store from context key %q",
			StoreContextKey,
		))
	}

	return s
}
