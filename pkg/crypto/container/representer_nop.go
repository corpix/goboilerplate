package container

var _ Representer = new(NopRepresenter)

type NopRepresenter struct{}

func (NopRepresenter) Encode(es []byte) []byte           { return es }
func (NopRepresenter) Decode(buf []byte) ([]byte, error) { return buf, nil }

//

func NewNopRepresenter() NopRepresenter {
	return NopRepresenter{}
}
