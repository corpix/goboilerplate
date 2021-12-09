package container

import (
	"encoding/json"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

//

var _ Serializer = new(JsonSerializer)

type JsonSerializer struct{}

func (JsonSerializer) Marshal(v interface{}) ([]byte, error) {
	buf, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal value to json")
	}

	return buf, nil
}

func (JsonSerializer) Unmarshal(buf []byte, v interface{}) error {
	err := json.Unmarshal(buf, v)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal value from json")
	}
	return nil
}

//

func NewJsonSerializer() JsonSerializer {
	return JsonSerializer{}
}
