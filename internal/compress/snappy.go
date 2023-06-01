package compress

import (
	"bytes"
	"io"

	"github.com/golang/snappy"
)

func Zip(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, EmptyData
	}

	var buffer bytes.Buffer
	writer := snappy.NewBufferedWriter(&buffer)
	_, err := writer.Write(data)
	if err != nil {
		_ = writer.Close()
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Unzip(data []byte) ([]byte, error) {
	if data == nil {
		return nil, EmptyData
	}

	reader := snappy.NewReader(bytes.NewReader(data))
	out, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return out, err
}
