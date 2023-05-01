package protocol

import (
	"encoding/binary"
	"errors"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/lightsocks/internal/cipher"
	"github.com/xmapst/lightsocks/internal/compress"
	"io"
	"math/rand"
)

type Packet struct {
	RandNu  int
	Payload []byte
}

const (
	payloadLen = uint32(4)
	randLen    = uint32(2)
	headerLen  = int(payloadLen + randLen)
	maxByte    = 1 << 24
)

var (
	packetEndian        = binary.BigEndian
	ErrIncompletePacket = errors.New("incomplete packet")
	ErrTooLargePacket   = errors.New("too large packet")
)

// Protocol format:
//
// * 0                 4      2
// * +-----------------+------+
// * |    body len     | rand |
// * +------+--------+--------+
// * |                        |
// * +                        +
// * |        body bytes      |
// * +                        +
// * |         ... ...        |
// * +-------------------------

func random(i int) int {
	if i <= 0 {
		return rand.Intn(99) + 1
	} else {
		return i % (rand.Intn(99) + 1)
	}
}

func Encode(key, bin []byte) ([]byte, error) {
	randNu := random(len(bin))
	// 压缩
	zipBin, err := compress.Zip(bin)
	if err != nil {
		return nil, err
	}
	// 加密
	encryptBuff := cipher.Encrypt(zipBin, key)

	msgLen := headerLen + len(encryptBuff)
	buffer := make([]byte, uint32(msgLen))

	// 添加头部信息
	packetEndian.PutUint32(buffer, uint32(len(encryptBuff)))
	packetEndian.PutUint16(buffer[payloadLen:], uint16(randNu))

	// 拷贝数据
	copy(buffer[headerLen:msgLen], encryptBuff)
	return buffer[:msgLen], nil
}

func UnPack(key, buf []byte) (*Packet, error) {
	if len(buf) < headerLen {
		return nil, ErrIncompletePacket
	}
	bodyLen := packetEndian.Uint32(buf[:payloadLen])
	randNu := packetEndian.Uint16(buf[payloadLen:headerLen])
	msgLen := headerLen + int(bodyLen)
	if len(buf) < msgLen {
		return nil, ErrIncompletePacket
	}
	// 解密
	decryptBuf := cipher.Decrypt(buf[headerLen:msgLen], key)
	// 解压
	unzipBuf, err := compress.Unzip(decryptBuf)
	if err != nil {
		logrus.Errorln(err.Error())
		return nil, err
	}
	packet := &Packet{
		RandNu:  int(randNu),
		Payload: unzipBuf,
	}
	return packet, nil
}

func ReadFull(key []byte, r io.Reader) (*Packet, error) {
	preBuff := make([]byte, headerLen)
	_, err := io.ReadFull(r, preBuff)
	if err != nil {
		return nil, err
	}
	bodyLen := packetEndian.Uint32(preBuff[:payloadLen])
	randNu := packetEndian.Uint16(preBuff[payloadLen:headerLen])
	if bodyLen > maxByte {
		return nil, ErrTooLargePacket
	}
	buf := make([]byte, bodyLen)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	// 解密
	decryptBuf := cipher.Decrypt(buf, key)
	// 解压
	unzipBuf, err := compress.Unzip(decryptBuf)
	if err != nil {
		logrus.Errorln(err.Error())
		return nil, err
	}
	packet := &Packet{
		RandNu:  int(randNu),
		Payload: unzipBuf,
	}
	return packet, nil
}
