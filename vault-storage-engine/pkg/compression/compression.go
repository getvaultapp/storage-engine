package compression

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pierrec/lz4/v4"
)

type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

// Compress uses LZ4 algorithm for fast compression with moderate compression module
func Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := lz4.NewWriter(&buf)
	_, err := writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to compression logic, %w", err)
	}
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close writer, %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses data using LZ4 algoritm
func Decompress(data []byte) ([]byte, error) {
	reader := lz4.NewReader(bytes.NewReader(data))
	var buf bytes.Buffer
	_, err := io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to appropriately Copy data, %w", err)
	}
	return buf.Bytes(), nil
}
