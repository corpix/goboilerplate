package middleware

import (
	echo "github.com/labstack/echo/v4"

	"git.backbone/corpix/goboilerplate/pkg/crypto"
	"git.backbone/corpix/goboilerplate/pkg/errors"
	"git.backbone/corpix/goboilerplate/pkg/server/session"
)

const SessionStoreContextKey = session.StoreContextKey

func NewSession(sc session.Config, rand crypto.Rand, options ...session.Option) echo.MiddlewareFunc {
	var (
		decryptErr      = crypto.ErrDecrypt{}
		formatErr       = crypto.ErrFormat{}
		invalidErr      = session.ErrInvalid{}
		incompatibleErr = session.ErrIncompatible{}
	)

	newStore := func(c echo.Context) (session.Store, error) {
		s, err := session.New(sc, rand, options...)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create session")
		}

		switch {
		case c.Request().Header.Get(echo.HeaderAuthorization) != "":
			return session.NewHeaderStore(echo.HeaderAuthorization, c, s), nil
		default:
			return session.NewCookieStore(*sc.Cookie, c, s), nil
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			l := c.Logger().(*Logger).Unwrap()

			store, err := newStore(c)
			if err != nil {
				return err
			}

			err, _ = store.Load()
			if err != nil {
				werr := errors.Wrap(err, "failed to load session")
				if errors.HasType(err, decryptErr) || errors.HasType(err, formatErr) {
					l.Warn().Err(werr).Msg("error while loading session")
					// it is ok to continue here because session was not loaded
					// and we still have our initially
				} else {
					return werr
				}
			}

			// NOTE: this may rewrite session inside store, be careful
			err = store.Session().Validate()
			if err != nil {
				werr := errors.Wrap(err, "failed to validate session")
				// if session is invalid or has wrong version then just drop it
				// otherwise fail the request
				if errors.HasType(err, invalidErr) || errors.HasType(err, incompatibleErr) {
					l.Warn().Err(werr).Msg("error while validating session, making a new one")

					store, err = newStore(c)
					if err != nil {
						return err
					}
				} else {
					return werr
				}
			}

			// at this point session is safe to use

			userSession := store.Session()

			if userSession.RefreshRequired() {
				l.Debug().
					Str("validAfter", userSession.Header().ValidAfter.String()).
					Str("validBefore", userSession.Header().ValidBefore.String()).
					Msg("session refresh required")
				userSession.Refresh()
			}

			c.Set(SessionStoreContextKey, store)

			//

			previousNonce := userSession.Header().Nonce
			err = next(c)
			if err != nil {
				return err
			}
			currentNonce := userSession.Header().Nonce

			//

			if currentNonce > previousNonce && !c.Response().Committed {
				l.Debug().
					Uint64("current-nonce", currentNonce).
					Uint64("previous-nonce", previousNonce).
					Msg("session nonce update")

				err = store.Save()
				if err != nil {
					return err
				}
			}

			return nil
		}
	}
}
