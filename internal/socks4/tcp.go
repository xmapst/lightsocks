package socks4

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/lightsocks/internal/constant"
)

type Proxy struct {
	id     uuid.UUID
	wg     *sync.WaitGroup
	conn   net.Conn
	server *constant.Server
}

func (p *Proxy) srcAddr() string {
	return p.conn.RemoteAddr().String()
}

/*
socks4 protocol
request
byte | 0  | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | ...  |
     |0x04|cmd| port  |     ip        |  user\0  |
reply
byte | 0  |  1   | 2 | 3 | 4 | 5 | 6 | 7|
     |0x00|status|       |              |
socks4a protocol
request
byte | 0  | 1 | 2 | 3 |4 | 5 | 6 | 7 | 8 | ... |...     |
     |0x04|cmd| port  |  0.0.0.x     |  user\0 |domain\0|
reply
byte | 0  |  1  | 2 | 3 | 4 | 5 | 6| 7 |
	 |0x00|staus| port  |    ip        |
*/

func (p *Proxy) New(wg *sync.WaitGroup, conf *constant.Server, uuid uuid.UUID, conn net.Conn) error {
	p.wg = wg
	p.id = uuid
	p.conn = conn
	p.server = conf
	return nil
}

func (p *Proxy) Handle(tcpIn chan<- *constant.TCPContext) error {
	header, err := p.handshake()
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	return p.processRequest(header, tcpIn)
}

func (p *Proxy) handshake() (addr string, err error) {
	var buf = make([]byte, 4096)
	var n int
	if n < 8 {
		n1, err := io.ReadAtLeast(p.conn, buf[n:], 8-n)
		if err != nil {
			logrus.Errorln(p.id, p.srcAddr(), ErrRequestRejected, err)
			return "", ErrRequestRejected
		}
		n += n1
	}
	buf = buf[1:n]
	command := buf[0]
	logrus.Infoln(p.id, p.srcAddr(), cmdMap[command])
	// command only support connect
	if command != CmdConnect {
		logrus.Errorln(p.id, p.srcAddr(), ErrRequestUnknownCode)
		return "", ErrRequestUnknownCode
	}
	user := p.readUntilNull(buf[7:])
	// TODO: VerifyUser
	logrus.Debugln(p.id, p.srcAddr(), user)

	// get port
	port := binary.BigEndian.Uint16(buf[1:3])

	// get ip
	ip := net.IP(buf[3:7])

	// NULL-terminated user string
	// jump to NULL character
	var j int
	for j = 7; j < n-1; j++ {
		if buf[j] == 0x00 {
			break
		}
	}

	host := ip.String()

	// socks4a
	// 0.0.0.x
	if ip[0] == 0x00 && ip[1] == 0x00 && ip[2] == 0x00 && ip[3] != 0x00 {
		j++
		var i = j

		// jump to the end of hostname
		for j = i; j < n-1; j++ {
			if buf[j] == 0x00 {
				break
			}
		}
		host = string(buf[i:j])
	}

	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

func (p *Proxy) processRequest(header string, tcpIn chan<- *constant.TCPContext) error {
	client, err := constant.UnmarshalIP(p.srcAddr())
	if err != nil {
		return err
	}
	source, err := constant.UnmarshalIP(p.conn.LocalAddr().String())
	if err != nil {
		return err
	}
	target, err := constant.UnmarshalIP(header)
	if err != nil {
		return err
	}
	tcpIn <- &constant.TCPContext{
		SrcConn: p.conn,
		Metadata: &constant.Metadata{
			ID:      p.id,
			NetWork: constant.TCP,
			Type:    constant.SOCKS4,
			Client:  client,
			Source:  source,
			Target:  target,
		},
		PreFn: func() {
			_, err = p.conn.Write([]byte{0x00, 0x5A, 0x00, 0x00, 0, 0, 0, 0})
			if err != nil {
				logrus.Errorln(p.id, p.srcAddr(), "write response error", err)
				return
			}
		},
		PostFn: func() {
			p.wg.Done()
		},
	}
	return nil
}

func (p *Proxy) readUntilNull(src []byte) string {
	buf := &bytes.Buffer{}
	for _, v := range src {
		if v == 0 {
			break
		}
		buf.WriteByte(v)
	}
	return buf.String()
}
