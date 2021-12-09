package session

import (
	"net/http"
	"strings"
	"time"

	echo "github.com/labstack/echo/v4"
)

var _ Store = &CookieStore{}

type CookieStore struct {
	config  CookieConfig
	context echo.Context
	session *Session
}

func (s *CookieStore) Load() (error, bool) {
	cookie, _ := s.context.Cookie(s.config.Name)
	// XXX: dang untyped errors
	// we can't infer the exact reason why cookie loading is failed
	// one way is "parse error message", but.. hey, thanks, we don't do it here
	if cookie == nil {
		return nil, false
	}

	err := s.session.Load([]byte(cookie.Value))
	if err != nil {
		return err, false
	}

	return nil, true
}

func (s *CookieStore) Save() error {
	buf, err := s.session.Save()
	if err != nil {
		return err
	}

	h := s.session.Header()

	s.setCookie(
		buf,
		h.ValidBefore.Sub(h.ValidAfter),
		h.ValidBefore,
	)

	return nil
}

func (s *CookieStore) Drop() error {
	s.setCookie(nil, -1, time.Unix(0, 0))

	return nil
}

func (s *CookieStore) setCookie(value []byte, maxAge time.Duration, expires time.Time) {
	domain := s.config.Domain
	if domain == "" {
		domain = s.context.Request().URL.Host
	}
	// net/http: invalid Cookie.Domain "xxx.localhost:4180"; dropping domain attribute
	domain = strings.Split(domain, ":")[0]

	s.context.SetCookie(&http.Cookie{
		Name:     s.config.Name,
		Value:    string(value),
		Path:     s.config.Path,
		Domain:   domain,
		MaxAge:   int(maxAge / time.Second),
		Expires:  expires,
		Secure:   *s.config.Secure,
		HttpOnly: *s.config.HTTPOnly,
		SameSite: SameSite[strings.ToLower(s.config.SameSite)],
	})
}

func (s *CookieStore) Session() *Session {
	return s.session
}

//

func NewCookieStore(c CookieConfig, ctx echo.Context, s *Session) *CookieStore {
	return &CookieStore{
		config:  c,
		context: ctx,
		session: s,
	}
}
