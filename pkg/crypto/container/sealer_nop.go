package container

var _ Sealer = new(NopSealer)

type NopSealer struct{}

func (ed NopSealer) Seal(buf []byte) ([]byte, error) { return buf, nil }
func (ed NopSealer) Open(enc []byte) ([]byte, error) { return enc, nil }

func NewNopSealer() NopSealer {
	return NopSealer{}
}
