package container

var _ Compressor = new(NopCompressor)

type NopCompressor struct{}

func (NopCompressor) Compress(buf []byte) []byte            { return buf }
func (NopCompressor) Decompress(buf []byte) ([]byte, error) { return buf, nil }

func NewNopCompressor() NopCompressor {
	return NopCompressor{}
}
