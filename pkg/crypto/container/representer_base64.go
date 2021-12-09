package container

import (
	"encoding/base64"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

var (
	base64Encoding = base64.URLEncoding
)

//

var _ Representer = new(Base64Representer)

type Base64Representer struct{}

func (Base64Representer) Encode(es []byte) []byte {
	buf := make([]byte, base64Encoding.EncodedLen(len(es)))
	base64Encoding.Encode(buf, es)

	return buf
}

func (Base64Representer) Decode(buf []byte) ([]byte, error) {
	es := make([]byte, base64Encoding.DecodedLen(len(buf)))
	n, err := base64Encoding.Decode(es, buf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode buf")
	}

	return es[:n], nil
}

//

func NewBase64Representer() Base64Representer {
	return Base64Representer{}
}
