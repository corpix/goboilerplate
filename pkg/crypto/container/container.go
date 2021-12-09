package container

import (
	"strings"
	"time"

	"git.backbone/corpix/goboilerplate/pkg/crypto"
	"git.backbone/corpix/goboilerplate/pkg/errors"
)

const (
	Version uint = 2

	SecretBoxType Type = "secretbox"
	JwtType       Type = "jwt"

	MsgPackSerializerType SerializerType = "msgpack"
	JsonSerializerType    SerializerType = "json"

	NopSealerType       SealerType = "nop"
	SecretBoxSealerType SealerType = "secretbox"

	NopCompressorType  CompressorType = "nop"
	ZstdCompressorType CompressorType = "zstd"

	NopRepresenterType    RepresenterType = "nop"
	Base64RepresenterType RepresenterType = "base64"
)

var (
	Types = map[Type]struct{}{
		SecretBoxType: {},
		JwtType:       {},
	}
	SerializerTypes = map[SerializerType]struct{}{
		MsgPackSerializerType: {},
		JsonSerializerType:    {},
	}
	SealerTypes = map[SealerType]struct{}{
		NopSealerType:       {},
		SecretBoxSealerType: {},
	}
	CompressorTypes = map[CompressorType]struct{}{
		NopCompressorType:  {},
		ZstdCompressorType: {},
	}
	RepresenterTypes = map[RepresenterType]struct{}{
		NopRepresenterType:    {},
		Base64RepresenterType: {},
	}
)

type (
	Nonce = uint64

	//

	Header struct {
		Version     uint
		Nonce       Nonce
		ValidAfter  time.Time
		ValidBefore time.Time
	}

	Payload      = map[PayloadKey]PayloadValue
	PayloadKey   uint
	PayloadValue []byte

	Data struct {
		Header
		Payload
	}

	//

	Container interface {
		Header() Header
		Payload() Payload
		Data() Data
		Encoder() Encoder

		Clean()
		Touch(Nonce)
		Validate() error
		Refresh(validAfter time.Time, validBefore time.Time)

		Set(key PayloadKey, value PayloadValue)
		Get(key PayloadKey) (PayloadValue, bool)
		Del(key PayloadKey) bool

		Save() ([]byte, error)
		Load([]byte) error
	}
	Type string

	//

	Serializer interface {
		Marshal(interface{}) ([]byte, error)
		Unmarshal([]byte, interface{}) error
	}
	SerializerType string

	//

	Compressor interface {
		Compress([]byte) []byte
		Decompress([]byte) ([]byte, error)
	}
	CompressorType string

	//

	Sealer interface {
		Seal([]byte) ([]byte, error)
		Open([]byte) ([]byte, error)
	}
	SealerType string

	//

	Representer interface {
		Encode([]byte) []byte
		Decode([]byte) ([]byte, error)
	}
	RepresenterType string

	//

	Encoder struct {
		Serializer
		Compressor
		Sealer
		Representer
	}
)

//

func New(c Config, rand crypto.Rand, validAfter time.Time, ttl time.Duration, payload Payload) (Container, error) {
	enc, err := NewEncoder(c, c.key, rand)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create encoder")
	}

	switch Type(strings.ToLower(c.Type)) {
	case SecretBoxType:
		return NewSecretBox(*c.SecretBox, enc, validAfter, ttl, payload)
	case JwtType:
		return NewJwt(*c.Jwt, enc, c.key, validAfter, ttl, payload)
	default:
		return nil, errors.Errorf("unsupported container type %q", c.Type)
	}
}

//

func NewEncoder(c Config, key []byte, rand crypto.Rand) (Encoder, error) {
	var (
		e   Encoder
		ent string
		t   string
		err error
	)

	//

	switch strings.ToLower(c.Serializer) {
	case string(MsgPackSerializerType):
		e.Serializer = NewMsgPackSerializer()
	case string(JsonSerializerType):
		e.Serializer = NewJsonSerializer()
	default:
		ent = "serializer"
		t = c.Serializer
		goto fail
	}
	if err != nil {
		return e, err
	}

	//

	switch strings.ToLower(c.Compressor) {
	case string(NopCompressorType):
		e.Compressor = NewNopCompressor()
	case string(ZstdCompressorType):
		e.Compressor = NewZstdCompressor()
	default:
		ent = "compressor"
		t = c.Compressor
		goto fail
	}

	//

	switch strings.ToLower(c.Sealer) {
	case string(NopSealerType):
		e.Sealer = NewNopSealer()
	case string(SecretBoxSealerType):
		e.Sealer, err = NewSecretBoxSealer(rand, key)
	default:
		ent = "sealer"
		t = c.Sealer
		goto fail
	}
	if err != nil {
		return e, err
	}

	//

	switch strings.ToLower(c.Representer) {
	case string(NopRepresenterType):
		e.Representer = NewNopRepresenter()
	case string(Base64RepresenterType):
		e.Representer = NewBase64Representer()
	default:
		ent = "representer"
		t = c.Representer
		goto fail
	}

	//

	return e, nil

fail:
	return e, errors.Errorf("unsupported %s %q", ent, t)
}
