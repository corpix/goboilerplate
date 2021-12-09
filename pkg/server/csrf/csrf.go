package csrf

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"net/url"
	"time"

	echo "github.com/labstack/echo/v4"
	"lukechampine.com/blake3"

	"git.backbone/corpix/goboilerplate/pkg/crypto"
	"git.backbone/corpix/goboilerplate/pkg/crypto/container"
	"git.backbone/corpix/goboilerplate/pkg/errors"
	serverErrors "git.backbone/corpix/goboilerplate/pkg/server/errors"
)

type (
	Payload    = container.Payload
	PayloadKey = container.PayloadKey

	Nonce = [8]byte
	Token = string
)

const (
	ContextKey    = "csrf"
	ParameterName = "csrf"

	SourceKey  PayloadKey = 0x10
	SubjectKey PayloadKey = 0x20
)

//

type CSRF struct {
	config  *Config
	encoder container.Encoder
	rand    crypto.Rand
}

func (t *CSRF) Checksum(nonce []byte, subject string) []byte {
	payload := make([]byte, len(nonce)+len(subject))
	copy(payload[copy(payload, nonce):], subject)
	sum := blake3.Sum256(payload)
	return sum[:]
}

func (t *CSRF) Sign(source string, subject string) (Token, error) {
	var nonceBuf Nonce
	_, err := t.rand.Read(nonceBuf[:])
	if err != nil {
		return "", err
	}
	nonce := binary.BigEndian.Uint64(nonceBuf[:])

	//

	box, err := t.newContainer()
	if err != nil {
		return "", err
	}
	box.Set(SourceKey, t.Checksum(nonceBuf[:], source))
	box.Set(SubjectKey, t.Checksum(nonceBuf[:], subject))
	box.Touch(nonce)

	buf, err := box.Save()
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func (t *CSRF) MustSign(source string, subject string) Token {
	token, err := t.Sign(source, subject)
	if err != nil {
		panic(err)
	}

	return token
}

func (t *CSRF) SignURL(source string, u *url.URL) (*url.URL, error) {
	token, err := t.Sign(source, u.String())
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set(t.config.ParameterName, token)

	uu := *u
	uu.RawQuery = q.Encode()

	return &uu, nil
}

func (t *CSRF) MustSignURL(source string, u *url.URL) *url.URL {
	uu, err := t.SignURL(source, u)
	if err != nil {
		panic(err)
	}

	return uu
}

func (t *CSRF) SignURLString(source string, u string) (string, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	uu, err = t.SignURL(source, uu)
	if err != nil {
		return "", err
	}

	return uu.String(), nil
}

func (t *CSRF) MustSignURLString(source string, u string) string {
	uu, err := t.SignURLString(source, u)
	if err != nil {
		panic(err)
	}

	return uu
}

func (t *CSRF) Validate(source string, subject string, token Token) error {
	box, err := t.Unpack(token)
	if err != nil {
		return err
	}

	err = box.Validate()
	if err != nil {
		return err
	}

	//

	var nonce Nonce
	binary.BigEndian.PutUint64(nonce[:], box.Header().Nonce)

	tokenSourceChecksum, _ := box.Get(SourceKey)
	tokenSubjectChecksum, _ := box.Get(SubjectKey)

	sourceChecksum := t.Checksum(nonce[:], source)
	subjectChecksum := t.Checksum(nonce[:], subject)

	if !bytes.Equal(tokenSourceChecksum, sourceChecksum) {
		return errors.Errorf(
			"token source checksum %x is not equals requested to the source checksum %x for input %q",
			tokenSourceChecksum, sourceChecksum, source,
		)
	}
	if !bytes.Equal(tokenSubjectChecksum, subjectChecksum) {
		return errors.Errorf(
			"token subject checksum %x is not equals requested to the subject checksum %x for input %q",
			tokenSubjectChecksum, subjectChecksum, subject,
		)
	}

	return nil
}

func (t *CSRF) ValidateURL(source string, u *url.URL) error {
	q := u.Query()

	token := q.Get(t.config.ParameterName)
	q.Del(t.config.ParameterName)

	uu := *u
	uu.RawQuery = q.Encode()

	return t.Validate(source, uu.String(), token)
}

func (t *CSRF) ValidateContext(c echo.Context) error {
	// NOTE: leave only path part bercause we don't care about scheme://hostname
	// if token is cryptographically secure (and this is convenient for signer to use in templates)
	// probably some additional strategy could be implemented in the future

	u := *c.Request().URL
	u.Scheme = ""
	u.Host = ""

	err := t.ValidateURL(c.RealIP(), &u)
	if err != nil {
		return serverErrors.NewError(
			http.StatusBadRequest, "CSRF token validation failed",
			err, nil,
		)
	}

	return nil
}

func (t *CSRF) Unpack(token Token) (container.Container, error) {
	box, err := t.newContainer()
	if err != nil {
		return nil, err
	}
	err = box.Load([]byte(token))
	if err != nil {
		return nil, err
	}

	return box, nil
}

func (t *CSRF) newContainer() (container.Container, error) {
	return container.NewSecretBox(
		container.SecretBoxConfig{},
		t.encoder,
		time.Now(),
		t.config.TTL,
		nil,
	)
}

//

func New(c Config, rand crypto.Rand) (*CSRF, error) {
	var err error

	enc, err := container.NewSecretBoxEncoder(rand, c.key)
	if err != nil {
		return nil, err
	}

	return &CSRF{
		config:  &c,
		encoder: enc,
		rand:    rand,
	}, nil
}
