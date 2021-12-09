package container

import (
	"git.backbone/corpix/goboilerplate/pkg/crypto"
	"git.backbone/corpix/goboilerplate/pkg/errors"
)

//

var _ Sealer = new(SecretBoxSealer)

type SecretBoxSealer struct {
	box *crypto.SecretBox
}

func (ed SecretBoxSealer) Seal(buf []byte) ([]byte, error) {
	nonce, err := ed.box.Nonce()
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate nonce")
	}

	return ed.box.Seal(nonce, buf), nil
}

func (ed SecretBoxSealer) Open(enc []byte) ([]byte, error) {
	buf, err := ed.box.Open(enc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open secret box")
	}

	return buf, nil
}

func NewSecretBoxSealer(rand crypto.Rand, key []byte) (SecretBoxSealer, error) {
	var e SecretBoxSealer

	//

	if crypto.SecretBoxKeySize != len(key) {
		return e, errors.Errorf(
			"invalid encryption key length, want %d, got %d",
			crypto.SecretBoxKeySize, len(key),
		)
	}
	boxKey := new(crypto.SecretBoxKey)
	copy(boxKey[:], key)

	//

	e.box = crypto.NewSecretBox(rand, boxKey)

	return e, nil
}
