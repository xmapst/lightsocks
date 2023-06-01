package net

import (
	"io"
	"sync"

	"github.com/xmapst/lightsocks/internal/protocol"
)

const (
	bufSize = 2 << 15
)

var bpool sync.Pool

func init() {
	bpool.New = func() interface{} {
		return make([]byte, bufSize)
	}
}
func bufferPoolGet() []byte {
	return bpool.Get().([]byte)
}
func bufferPoolPut(b []byte) {
	bpool.Put(b)
}

// SecureTCPConn 加密传输的 TCP Socket
type SecureTCPConn struct {
	io.ReadWriteCloser
}

// EncodeWrite 把放在bs里的数据加密后立即全部写入输出流
func (secureSocket *SecureTCPConn) EncodeWrite(token, bs []byte) (int, error) {
	// 加密
	data, err := protocol.Encode(token, bs)
	if err != nil {
		return 0, err
	}
	return secureSocket.Write(data)
}

func (secureSocket *SecureTCPConn) EncodeCopy(token []byte, dst io.ReadWriteCloser) error {
	buf := bufferPoolGet()
	defer bufferPoolPut(buf)
	for {
		readCount, errRead := secureSocket.Read(buf)
		if errRead != nil {
			if errRead != io.EOF {
				return errRead
			} else {
				return nil
			}
		}
		if readCount > 0 {
			_, errWrite := (&SecureTCPConn{
				ReadWriteCloser: dst,
			}).EncodeWrite(token, buf[0:readCount])
			if errWrite != nil {
				return errWrite
			}
		}
	}
}
