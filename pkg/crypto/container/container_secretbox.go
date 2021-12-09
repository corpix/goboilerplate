package container

import (
	"sync"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/crypto"
)

var (
	_ Container = new(SecretBoxContainer)
)

type (
	SecretBoxContainer struct {
		lock    *sync.RWMutex
		config  SecretBoxConfig
		encoder Encoder
		data    SecretBoxContainerData
	}
	SecretBoxContainerData struct {
		Header  Header
		Payload Payload
	}
)

//

func (s *SecretBoxContainer) Header() Header {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.data.Header
}

func (s *SecretBoxContainer) Payload() Payload {
	s.lock.RLock()
	defer s.lock.RUnlock()

	res := make(Payload, len(s.data.Payload))
	for k, v := range s.data.Payload {
		res[k] = v
	}
	return res
}

func (s *SecretBoxContainer) Data() Data {
	s.lock.RLock()
	defer s.lock.RUnlock()

	d := Data{Header: s.data.Header}
	d.Payload = make(Payload, len(s.data.Payload))
	for k, v := range s.data.Payload {
		d.Payload[k] = v
	}

	return d
}

func (s *SecretBoxContainer) Encoder() Encoder { return s.encoder }

//

func (s *SecretBoxContainer) Clean() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data.Payload = Payload{}
}

func (s *SecretBoxContainer) Touch(n Nonce) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if n == 0 {
		s.data.Header.Nonce++
	} else {
		s.data.Header.Nonce = n
	}
}

func (s *SecretBoxContainer) Validate() error {
	t := time.Now()

	if s.data.Header.Version != Version {
		return ErrIncompatible{
			Subject: "version",
			Meta: []interface{}{
				"want", Version,
				"got", s.data.Header.Version,
			},
		}
	}

	if !t.After(s.data.Header.ValidAfter) {
		return ErrInvalid{
			Subject: "not yet valid",
			Meta: []interface{}{
				"valid after", s.data.Header.ValidAfter,
				"now", t,
			},
		}
	}

	if !t.Before(s.data.Header.ValidBefore) {
		return ErrInvalid{
			Subject: "expired",
			Meta: []interface{}{
				"valid before", s.data.Header.ValidBefore,
				"now", t,
			},
		}
	}

	return nil
}

func (s *SecretBoxContainer) Refresh(validAfter time.Time, validBefore time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data.Header.Nonce++
	s.data.Header.ValidAfter = validAfter
	s.data.Header.ValidBefore = validBefore
}

//

func (s *SecretBoxContainer) Get(key PayloadKey) (PayloadValue, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	v, ok := s.data.Payload[key]
	if !ok {
		return nil, false
	}

	return v, true
}

func (s *SecretBoxContainer) Set(key PayloadKey, value PayloadValue) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.data.Payload[key] = value
}

func (s *SecretBoxContainer) Del(key PayloadKey) bool {
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

func (s *SecretBoxContainer) Save() ([]byte, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	buf, err := s.encoder.Marshal(s.data)
	if err != nil {
		return nil, err
	}
	enc, err := s.encoder.Seal(s.encoder.Compress(buf))
	if err != nil {
		return nil, err
	}

	return s.encoder.Encode(enc), nil
}

func (s *SecretBoxContainer) Load(buf []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	enc, err := s.encoder.Decode(buf)
	if err != nil {
		return err
	}
	dec, err := s.encoder.Open(enc)
	if err != nil {
		return err
	}
	raw, err := s.encoder.Decompress(dec)
	if err != nil {
		return err
	}

	err = s.encoder.Unmarshal(raw, &s.data)
	if err != nil {
		return err
	}

	return nil
}

//

func NewSecretBox(c SecretBoxConfig, enc Encoder, validAfter time.Time, ttl time.Duration, payload Payload) (*SecretBoxContainer, error) {
	if payload == nil {
		payload = make(Payload)
	}

	//

	p := make(Payload, len(payload))
	for k, v := range payload {
		p[k] = v
	}

	return &SecretBoxContainer{
		lock:    &sync.RWMutex{},
		config:  c,
		encoder: enc,
		data: SecretBoxContainerData{
			Header: Header{
				Version:     Version,
				ValidAfter:  validAfter,
				ValidBefore: validAfter.Add(ttl),
			},
			Payload: p,
		},
	}, nil
}

func NewSecretBoxEncoder(rand crypto.Rand, key []byte) (Encoder, error) {
	var (
		e   Encoder
		err error
	)

	e.Serializer = NewMsgPackSerializer()
	e.Compressor = NewZstdCompressor()
	e.Sealer, err = NewSecretBoxSealer(rand, key)
	if err != nil {
		return e, err
	}
	e.Representer = NewBase64Representer()

	return e, nil
}
