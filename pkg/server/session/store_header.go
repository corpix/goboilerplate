package session

import (
	"strings"

	echo "github.com/labstack/echo/v4"
)

var _ Store = &HeaderStore{}

type HeaderStore struct {
	name    string
	context echo.Context
	session *Session
}

func (s *HeaderStore) Load() (error, bool) {
	header := s.context.Request().Header.Get(s.name)
	if header == "" {
		return nil, false
	}

	// XXX: cover values like "Bearer XXXX"
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 {
		header = parts[1]
	}

	//

	err := s.session.Load([]byte(header))
	if err != nil {
		return err, false
	}

	return nil, true
}

func (s *HeaderStore) Save() error {
	// no implementation required because we are working with request header
	return nil
}

func (s *HeaderStore) Drop() error {
	// no implementation required because we are working with request header
	return nil
}

func (s *HeaderStore) Session() *Session {
	return s.session
}

//

func NewHeaderStore(name string, ctx echo.Context, s *Session) *HeaderStore {
	return &HeaderStore{
		name:    name,
		context: ctx,
		session: s,
	}
}
