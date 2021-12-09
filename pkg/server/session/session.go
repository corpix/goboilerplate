package session

import (
	"net/http"
	"strings"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/crypto"
	"git.backbone/corpix/goboilerplate/pkg/crypto/container"
	"git.backbone/corpix/goboilerplate/pkg/meta"
)

const (
	StoreContextKey = "session-store"

	Version = container.Version

	SameSiteDefault = "default"
	SameSiteLax     = "lax"
	SameSiteStrict  = "strict"
	SameSiteNone    = "none"
)

var (
	Name     = "_" + strings.ReplaceAll(meta.Name, "-", "_")
	SameSite = map[string]http.SameSite{
		SameSiteDefault: http.SameSiteDefaultMode,
		SameSiteLax:     http.SameSiteLaxMode,
		SameSiteStrict:  http.SameSiteStrictMode,
		SameSiteNone:    http.SameSiteNoneMode,
	}
)

type (
	Container       = container.Container
	ContainerConfig = container.Config

	Header       = container.Header
	Payload      = container.Payload
	PayloadKey   = container.PayloadKey
	PayloadValue = container.PayloadValue
	Data         = container.Data
	Encoder      = container.Encoder

	ErrIncompatible = container.ErrIncompatible
	ErrInvalid      = container.ErrInvalid

	Session struct {
		config     Config
		container  Container
		rand       crypto.Rand
		options    []Option
		validators []Validator
	}
)

//

func (s *Session) Header() Header   { return s.container.Header() }
func (s *Session) Payload() Payload { return s.container.Payload() }
func (s *Session) Data() Data       { return s.container.Data() }
func (s *Session) Encoder() Encoder { return s.container.Encoder() }

func (s *Session) Clean() { s.container.Clean() }
func (s *Session) Touch() { s.container.Touch(0) }

func (s *Session) Validate() error {
	err := s.container.Validate()
	if err != nil {
		return err
	}

	for _, validate := range s.validators {
		err = validate(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Session) RefreshRequired() bool {
	return time.Now().After(s.container.Header().ValidAfter.Add(s.config.Refresh))
}

func (s *Session) Refresh() {
	t := time.Now()
	s.container.Refresh(t, t.Add(s.config.MaxAge))
}

//

func (s *Session) Get(key PayloadKey) (PayloadValue, bool) {
	return s.container.Get(key)
}

func (s *Session) GetString(key PayloadKey) (string, bool) {
	bytes, ok := s.container.Get(key)
	return string(bytes), ok
}

func (s *Session) Set(key PayloadKey, value PayloadValue) {
	s.container.Set(key, value)
	s.Refresh()
}

func (s *Session) SetString(key PayloadKey, value string) {
	s.container.Set(key, []byte(value))
	s.Refresh()
}

func (s *Session) Del(key PayloadKey) bool {
	ok := s.container.Del(key)
	if ok {
		s.Refresh()
	}
	return ok
}

//

func (s *Session) Save() ([]byte, error)  { return s.container.Save() }
func (s *Session) Load(buf []byte) error  { return s.container.Load(buf) }
func (s *Session) New() (*Session, error) { return New(s.config, s.rand, s.options...) }

//

type (
	Option    = func(*Session)
	Validator = func(*Session) error
)

func WithPayloadKey(key PayloadKey, value PayloadValue) Option {
	return func(s *Session) {
		s.Set(key, value)
	}
}

func WithValidator(validate Validator) Option {
	return func(s *Session) {
		s.validators = append(s.validators, validate)
	}
}

//

func New(c Config, rand crypto.Rand, options ...Option) (*Session, error) {
	cont, err := container.New(*c.Container, rand, time.Now(), c.MaxAge, nil)
	if err != nil {
		return nil, err
	}

	sn := &Session{
		config:    c,
		container: cont,
		rand:      rand,
		options:   options,
	}

	for _, option := range options {
		option(sn)
	}

	return sn, nil
}
