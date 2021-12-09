package container

import (
	"github.com/klauspost/compress/zstd"

	"git.backbone/corpix/goboilerplate/pkg/errors"
)

var (
	zstdCompressor   *zstd.Encoder
	zstdDecompressor *zstd.Decoder
)

func init() {
	var err error

	zstdCompressor, err = zstd.NewWriter(nil)
	if err != nil {
		panic(errors.Wrap(err, "failed to construct zstd writer"))
	}
	zstdDecompressor, err = zstd.NewReader(nil)
	if err != nil {
		panic(errors.Wrap(err, "failed to construct zstd reader"))
	}
}

//

var _ Compressor = new(ZstdCompressor)

type ZstdCompressor struct{}

func (ZstdCompressor) Compress(buf []byte) []byte {
	return zstdCompressor.EncodeAll(buf, make([]byte, 0, len(buf)))
}

func (ZstdCompressor) Decompress(buf []byte) ([]byte, error) {
	return zstdDecompressor.DecodeAll(buf, nil)
}

func NewZstdCompressor() ZstdCompressor {
	return ZstdCompressor{}
}
