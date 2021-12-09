package container

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strings"
	"sync"
	"unsafe"

	jwt "github.com/cristalhq/jwt/v4"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

type (
	jwtMarshalerIndex = map[uintptr]*JwtMarshaler
	jwtMarshalerPool  struct {
		sync.RWMutex
		ms jwtMarshalerIndex
	}
)

var jwtMarshalerPoolDefault = newJwtMarshalerPool()

func (p *jwtMarshalerPool) Get(algoName string, key []byte) (*JwtMarshaler, error) {
	k := uintptr(unsafe.Pointer(&key[0]))

	p.RLock()
	m := p.ms[k]
	p.RUnlock()

	if m == nil {
		p.Lock()
		defer p.Unlock()

		var err error

		m = p.ms[k]
		if m == nil {
			m, err = NewJwtMarshaler(algoName, key)
			if err != nil {
				return nil, err
			}

			p.ms[k] = m
		}
	}

	return m, nil
}

func newJwtMarshalerPool() *jwtMarshalerPool {
	return &jwtMarshalerPool{
		ms: jwtMarshalerIndex{},
	}
}

//

type JwtMarshaler struct {
	builder  *jwt.Builder
	verifier jwt.Verifier
}

func (j JwtMarshaler) Marshal(v interface{}) ([]byte, error) {
	t, err := j.builder.Build(v)
	if err != nil {
		return nil, err
	}

	return t.Bytes(), nil
}

func (j JwtMarshaler) Unmarshal(buf []byte, v interface{}) error {
	vv, ok := v.(*JwtContainerData)
	if !ok {
		return errors.Errorf("expected *JwtContainerData, got %T", v)
	}
	if vv == nil {
		return errors.New("got nil pointer as target to write into")
	}

	t, err := jwt.Parse(buf, j.verifier)
	if err != nil {
		return err
	}

	err = json.Unmarshal(t.Claims(), &vv)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal value from jwt")
	}

	return nil
}

//

func NewJwtMarshaler(algoName string, key []byte) (*JwtMarshaler, error) {
	var (
		sr  JwtMarshaler
		jsr jwt.Signer
		jvr jwt.Verifier
		err error
	)

	algo, ok := JwtAlgorithms[strings.ToLower(algoName)]
	if !ok {
		return nil, errors.Errorf("unsupported jwt algorithm %q", algoName)
	}

	switch algo {
	case JwtAlgorithmHS256, JwtAlgorithmHS384, JwtAlgorithmHS512:
		jsr, err = jwt.NewSignerHS(algo, key)
		if err != nil {
			return nil, err
		}

		jvr, err = jwt.NewVerifierHS(algo, key)
		if err != nil {
			return nil, err
		}
	case JwtAlgorithmES256, JwtAlgorithmES384, JwtAlgorithmES512:
		block, _ := pem.Decode(key)
		ecdsaPrivateKey, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ecdsaPublicKey := ecdsaPrivateKey.Public().(*ecdsa.PublicKey)

		jsr, err = jwt.NewSignerES(algo, ecdsaPrivateKey)
		if err != nil {
			return nil, err
		}

		jvr, err = jwt.NewVerifierES(algo, ecdsaPublicKey)
		if err != nil {
			return nil, err
		}
	case JwtAlgorithmPS256, JwtAlgorithmPS384, JwtAlgorithmPS512:
		block, _ := pem.Decode(key)
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.Errorf("private key is not *rsa.PrivateKey, it is %T", privateKey)
		}
		rsaPublicKey := rsaPrivateKey.Public().(*rsa.PublicKey)

		jsr, err = jwt.NewSignerPS(algo, rsaPrivateKey)
		if err != nil {
			return nil, err
		}

		jvr, err = jwt.NewVerifierPS(algo, rsaPublicKey)
		if err != nil {
			return nil, err
		}
	case JwtAlgorithmRS256, JwtAlgorithmRS384, JwtAlgorithmRS512:
		block, _ := pem.Decode(key)
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rsaPrivateKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.Errorf("private key is not *rsa.PrivateKey, it is %T", privateKey)
		}
		rsaPublicKey := rsaPrivateKey.Public().(*rsa.PublicKey)

		jsr, err = jwt.NewSignerRS(algo, rsaPrivateKey)
		if err != nil {
			return nil, err
		}

		jvr, err = jwt.NewVerifierRS(algo, rsaPublicKey)
		if err != nil {
			return nil, err
		}
	case JwtAlgorithmEdDSA:
		block, _ := pem.Decode(key)
		privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ed25519PrivateKey, ok := privateKey.(ed25519.PrivateKey)
		if !ok {
			return nil, errors.Errorf("private key is not ed25519.PrivateKey, it is %T", privateKey)
		}
		ed25519PublicKey := ed25519PrivateKey.Public().(ed25519.PublicKey)

		jsr, err = jwt.NewSignerEdDSA(ed25519PrivateKey)
		if err != nil {
			return nil, err
		}

		jvr, err = jwt.NewVerifierEdDSA(ed25519PublicKey)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("unsupported jwt marshaling algorithm %q", algo)
	}

	sr.builder = jwt.NewBuilder(jsr)
	sr.verifier = jvr

	return &sr, nil
}
