package container

import (
	"strings"
	"sync"
	"time"

	jwt "github.com/cristalhq/jwt/v4"

	"git.backbone/corpix/goboilerplate/pkg/crypto"
)

var (
	_ Container = new(JwtContainer)
)

type (
	JwtContainer struct {
		lock   *sync.RWMutex
		config JwtConfig
		// encoder exists only to conform profile encoding
		// (which is json)
		// jwt token encoding is handled by JwtMarshaler
		encoder Encoder
		jwt     *JwtMarshaler
		data    JwtContainerData
	}

	// jwt sucks, so we need some strange data distribution

	JwtContainerHeader = jwt.Header
	JwtContainerData   struct {
		jwt.RegisteredClaims

		Version uint  `json:"version"`
		Nonce   Nonce `json:"nonce"`

		Payload Payload `json:"payload,omitempty"`
	}

	JwtAlgorithm = jwt.Algorithm
)

const (
	JwtAlgorithmEdDSA JwtAlgorithm = jwt.EdDSA

	JwtAlgorithmHS256 JwtAlgorithm = jwt.HS256
	JwtAlgorithmHS384 JwtAlgorithm = jwt.HS384
	JwtAlgorithmHS512 JwtAlgorithm = jwt.HS512

	JwtAlgorithmRS256 JwtAlgorithm = jwt.RS256
	JwtAlgorithmRS384 JwtAlgorithm = jwt.RS384
	JwtAlgorithmRS512 JwtAlgorithm = jwt.RS512

	JwtAlgorithmES256 JwtAlgorithm = jwt.ES256
	JwtAlgorithmES384 JwtAlgorithm = jwt.ES384
	JwtAlgorithmES512 JwtAlgorithm = jwt.ES512

	JwtAlgorithmPS256 JwtAlgorithm = jwt.PS256
	JwtAlgorithmPS384 JwtAlgorithm = jwt.PS384
	JwtAlgorithmPS512 JwtAlgorithm = jwt.PS512
)

var (
	JwtAlgorithms = map[string]JwtAlgorithm{
		strings.ToLower(string(JwtAlgorithmEdDSA)): JwtAlgorithmEdDSA,

		strings.ToLower(string(JwtAlgorithmHS256)): JwtAlgorithmHS256,
		strings.ToLower(string(JwtAlgorithmHS384)): JwtAlgorithmHS384,
		strings.ToLower(string(JwtAlgorithmHS512)): JwtAlgorithmHS512,

		strings.ToLower(string(JwtAlgorithmRS256)): JwtAlgorithmRS256,
		strings.ToLower(string(JwtAlgorithmRS384)): JwtAlgorithmRS384,
		strings.ToLower(string(JwtAlgorithmRS512)): JwtAlgorithmRS512,

		strings.ToLower(string(JwtAlgorithmES256)): JwtAlgorithmES256,
		strings.ToLower(string(JwtAlgorithmES384)): JwtAlgorithmES384,
		strings.ToLower(string(JwtAlgorithmES512)): JwtAlgorithmES512,

		strings.ToLower(string(JwtAlgorithmPS256)): JwtAlgorithmPS256,
		strings.ToLower(string(JwtAlgorithmPS384)): JwtAlgorithmPS384,
		strings.ToLower(string(JwtAlgorithmPS512)): JwtAlgorithmPS512,
	}
)

//

func (s *JwtContainer) Header() Header {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return Header{
		Version:     s.data.Version,
		Nonce:       s.data.Nonce,
		ValidAfter:  s.data.IssuedAt.Time,
		ValidBefore: s.data.ExpiresAt.Time,
	}
}

func (s *JwtContainer) Payload() Payload {
	s.lock.RLock()
	defer s.lock.RUnlock()

	res := make(Payload, len(s.data.Payload))
	for k, v := range s.data.Payload {
		res[k] = v
	}
	return res
}

func (s *JwtContainer) Data() Data {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return Data{
		Header:  s.Header(),
		Payload: s.Payload(),
	}
}

func (s *JwtContainer) Encoder() Encoder { return s.encoder }

//

func (s *JwtContainer) Clean() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data.Payload = Payload{}
}

func (s *JwtContainer) Touch(n Nonce) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if n == 0 {
		s.data.Nonce++
	} else {
		s.data.Nonce = n
	}
}

func (s *JwtContainer) Validate() error {
	t := time.Now()

	if s.data.Version != Version {
		return ErrIncompatible{
			Subject: "version",
			Meta: []interface{}{
				"want", Version,
				"got", s.data.Version,
			},
		}
	}

	if !t.After(s.data.IssuedAt.Time) {
		return ErrInvalid{
			Subject: "not yet valid",
			Meta: []interface{}{
				"valid after", s.data.IssuedAt.Time,
				"now", t,
			},
		}
	}

	if !t.Before(s.data.ExpiresAt.Time) {
		return ErrInvalid{
			Subject: "expired",
			Meta: []interface{}{
				"valid before", s.data.ExpiresAt.Time,
				"now", t,
			},
		}
	}

	return nil
}

func (s *JwtContainer) Refresh(validAfter time.Time, validBefore time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data.Nonce++
	s.data.NotBefore.Time = validAfter
	s.data.IssuedAt.Time = validAfter
	s.data.ExpiresAt.Time = validBefore
}

//

func (s *JwtContainer) Get(key PayloadKey) (PayloadValue, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v, ok := s.data.Payload[key]
	if !ok {
		return nil, false
	}

	return v, true
}

func (s *JwtContainer) Set(key PayloadKey, value PayloadValue) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data.Payload[key] = value
}

func (s *JwtContainer) Del(key PayloadKey) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.data.Payload[key]; ok {
		delete(s.data.Payload, key)
		return ok
	} else {
		return ok
	}
}

//

func (s *JwtContainer) Save() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.jwt.Marshal(s.data)
}

func (s *JwtContainer) Load(buf []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.jwt.Unmarshal(buf, &s.data)
	if err != nil {
		switch err {
		case jwt.ErrAlgorithmMismatch:
			return crypto.ErrFormat{Msg: err.Error()}
		case jwt.ErrInvalidFormat:
			return crypto.ErrFormat{Msg: err.Error()}
		case jwt.ErrInvalidSignature:
			return crypto.ErrDecrypt{Msg: err.Error()}
		default:
			return err
		}
	}

	return nil
}

//

func NewJwt(c JwtConfig, enc Encoder, key []byte, validAfter time.Time, ttl time.Duration, payload Payload) (*JwtContainer, error) {
	sr, err := jwtMarshalerPoolDefault.Get(c.Algo, key)
	if err != nil {
		return nil, err
	}

	if payload == nil {
		payload = make(Payload)
	}

	//

	p := make(Payload, len(payload))
	for k, v := range payload {
		p[k] = v
	}

	return &JwtContainer{
		lock:    &sync.RWMutex{},
		config:  c,
		encoder: enc,
		jwt:     sr,
		data: JwtContainerData{
			Version: Version,
			RegisteredClaims: jwt.RegisteredClaims{
				NotBefore: &jwt.NumericDate{Time: validAfter},
				IssuedAt:  &jwt.NumericDate{Time: validAfter},
				ExpiresAt: &jwt.NumericDate{Time: validAfter.Add(ttl)},
			},
			Payload: p,
		},
	}, nil
}

func NewJwtEncoder() Encoder {
	var e Encoder

	e.Serializer = NewJsonSerializer()
	e.Compressor = NewNopCompressor()
	e.Sealer = NewNopSealer()
	e.Representer = NewNopRepresenter()

	return e
}
