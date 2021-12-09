package container

import (
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"git.backbone/corpix/goboilerplate/pkg/errors"
	"git.backbone/corpix/goboilerplate/pkg/reflect"
)

//

type Config struct {
	Type string `yaml:"type"`

	Serializer  string `yaml:"serializer"`
	Sealer      string `yaml:"sealer"`
	Compressor  string `yaml:"compressor"`
	Representer string `yaml:"representer"`

	Key     string `yaml:"key"`
	KeyFile string `yaml:"key-file"`
	key     []byte

	SecretBox *SecretBoxConfig `yaml:"secretbox"`
	Jwt       *JwtConfig       `yaml:"jwt"`
}

func (c *Config) Default() {
loop:
	for {
		switch {
		case c.Type == "":
			c.Type = string(SecretBoxType)
		case c.Serializer == "":
			switch Type(c.Type) {
			case SecretBoxType:
				c.Serializer = string(MsgPackSerializerType)
			case JwtType:
				// serialization is handled out of bound
				// but this serializer is used to serialize profile data too
				c.Serializer = string(JsonSerializerType)
			default:
				c.Serializer = string(MsgPackSerializerType)
			}
		case c.Sealer == "":
			switch Type(c.Type) {
			case SecretBoxType:
				c.Sealer = string(SecretBoxSealerType)
			case JwtType:
				// sealing is handled out of bound
				c.Sealer = string(NopSealerType)
			default:
				c.Sealer = string(SecretBoxSealerType)
			}
		case c.Compressor == "":
			switch Type(c.Type) {
			case SecretBoxType:
				c.Compressor = string(ZstdCompressorType)
			case JwtType:
				// serialization is handled out of bound
				// also there is no compression in jwt
				c.Compressor = string(NopCompressorType)
			default:
				c.Compressor = string(ZstdCompressorType)
			}
		case c.Representer == "":
			switch Type(c.Type) {
			case SecretBoxType:
				c.Representer = string(Base64RepresenterType)
			case JwtType:
				// representation is handled out of bound
				c.Representer = string(NopRepresenterType)
			default:
				c.Representer = string(Base64RepresenterType)
			}
		case c.SecretBox == nil:
			c.SecretBox = &SecretBoxConfig{}
		case c.Jwt == nil:
			c.Jwt = &JwtConfig{}
		default:
			break loop
		}
	}
}

func (c *Config) Expand() error {
	var err error

	if c.KeyFile != "" {
		c.key, err = ioutil.ReadFile(c.KeyFile)
		if err != nil {
			return errors.Wrapf(
				err, "failed to load key-file: %q",
				c.KeyFile,
			)
		}
	} else {
		c.key = []byte(c.Key)
	}

	return nil
}

func (c *Config) Validate() error {
	var (
		// types index
		ts interface{}
		// given type
		t string
		// corresponding entity (configuration key) to mention in error
		e string
	)

	if _, ok := Types[Type(c.Type)]; !ok {
		ts = Types
		t = c.Type
		e = "type"
		goto fail
	}
	if _, ok := SerializerTypes[SerializerType(c.Serializer)]; !ok {
		ts = SerializerTypes
		t = c.Serializer
		e = "serializer"
		goto fail
	}
	if _, ok := SealerTypes[SealerType(c.Sealer)]; !ok {
		ts = SealerTypes
		t = c.Sealer
		e = "sealer"
		goto fail
	}
	if _, ok := CompressorTypes[CompressorType(c.Compressor)]; !ok {
		ts = CompressorTypes
		t = c.Compressor
		e = "compressor"
		goto fail
	}
	if _, ok := RepresenterTypes[RepresenterType(c.Representer)]; !ok {
		ts = RepresenterTypes
		t = c.Representer
		e = "representer"
		goto fail
	}

	switch Type(c.Type) {
	case JwtType:
		if c.Serializer != string(JsonSerializerType) {
			return errors.Errorf(
				"unexpected serializer %q for %q, expected %q",
				c.Serializer, c.Type, JsonSerializerType,
			)
		}
		if c.Sealer != string(NopSealerType) {
			return errors.Errorf(
				"unexpected sealer %q for %q, expected %q, sealing is handled out of bound",
				c.Sealer, c.Type, NopSealerType,
			)
		}
		if c.Compressor != string(NopCompressorType) {
			return errors.Errorf(
				"unexpected compressor %q for %q, expected %q, there is no compression in jwt",
				c.Compressor, c.Type, NopCompressorType,
			)
		}
		if c.Representer != string(NopRepresenterType) {
			return errors.Errorf(
				"unexpected representer %q for %q, expected %q, representation is handled out of bound",
				c.Representer, c.Type, NopRepresenterType,
			)
		}
	}

	if c.Key != "" && c.KeyFile != "" {
		return errors.New("either key or key-file must be defined, not both")
	}
	if c.Key == "" && c.KeyFile == "" {
		return errors.New("either key or key-file must be defined")
	}
	if len(c.key) == 0 {
		return errors.New("key length should be greater than zero")
	}

	return nil

fail:
	// enumerate available types and build error message
	n := 0
	mk := reflect.IndirectValue(reflect.ValueOf(ts)).MapKeys()
	available := make([]string, len(mk))
	for _, v := range mk {
		available[n] = fmt.Sprint(v)
		n++
	}
	sort.Strings(available)

	return errors.Errorf(
		"unexpected container %s %q, expected one of: %q",
		e, t, available,
	)
}

//

type SecretBoxConfig struct{}

//

type JwtConfig struct {
	Algo string `yaml:"algo"`
}

func (c *JwtConfig) Default() {
loop:
	for {
		switch {
		case c.Algo == "":
			c.Algo = string(JwtAlgorithmHS256)
		default:
			break loop
		}
	}
}

func (c *JwtConfig) Validate() error {
	if _, ok := JwtAlgorithms[strings.ToLower(c.Algo)]; !ok {
		algos := make([]string, len(JwtAlgorithms))
		n := 0
		for k := range JwtAlgorithms {
			algos[n] = k
			n++
		}
		sort.Strings(algos)
		return errors.Errorf("unsupported algorithm %q, should be one of: %v", c.Algo, algos)
	}
	return nil
}
