package container

import (
	msgpack "github.com/vmihailenco/msgpack/v5"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

var _ Serializer = new(MsgPackSerializer)

//

type MsgPackSerializer struct{}

func (MsgPackSerializer) Marshal(v interface{}) ([]byte, error) {
	buf, err := msgpack.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal value to msgpack")
	}

	return buf, nil
}

func (MsgPackSerializer) Unmarshal(buf []byte, v interface{}) error {
	err := msgpack.Unmarshal(buf, v)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal value from msgpack")
	}
	return nil
}

//

func NewMsgPackSerializer() MsgPackSerializer {
	return MsgPackSerializer{}
}
