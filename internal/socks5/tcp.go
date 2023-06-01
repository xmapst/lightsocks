package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
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
	Udp    string
}

type DialFunc func(network, addr string) (net.Conn, error)

func (p *Proxy) srcAddr() string {
	return p.conn.RemoteAddr().String()
}

/*
socks5 protocol
initial
byte | 0  |   1    | 2 | ...... | n |
     |0x05|num auth|  auth methods  |
reply
byte | 0  |  1  |
     |0x05| auth|
username/password auth request
byte | 0  |  1         |          |     1 byte   |          |
     |0x01|username_len| username | password_len | password |
username/password auth reponse
byte | 0  | 1    |
     |0x01|status|
request
byte | 0  | 1 | 2  |   3    | 4 | .. | n-2 | n-1| n |
     |0x05|cmd|0x00|addrtype|      addr    |  port  |
response
byte |0   |  1   | 2  |   3    | 4 | .. | n-2 | n-1 | n |
     |0x05|status|0x00|addrtype|     addr     |  port   |
*/

func (p *Proxy) New(wg *sync.WaitGroup, conf *constant.Server, uuid uuid.UUID, conn net.Conn) error {
	p.wg = wg
	p.id = uuid
	p.conn = conn
	p.server = conf
	return nil
}

func (p *Proxy) Handle(tcpIn chan<- *constant.TCPContext) error {
	if err := p.handshake(); err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	return p.processRequest(tcpIn)
}

func (p *Proxy) handshake() error {
	var buf = make([]byte, 4096)
	var n int
	// read auth methods
	if n < 2 {
		n1, err := io.ReadAtLeast(p.conn, buf, 1)
		if err != nil {
			logrus.Errorln(p.id, p.srcAddr(), err)
			return err
		}
		n += n1
	}
	l := int(buf[1])
	if n != (l + 2) {
		// read remains data
		n1, err := io.ReadFull(p.conn, buf[n:l+2+1])
		if err != nil {
			logrus.Errorln(p.id, p.srcAddr(), err)
			return err
		}
		n += n1
	}

	// Default: no auth required
	_, _ = p.conn.Write([]byte{0x05, 0x00})
	return nil

	// TODO: VerifyUser
	// hasPassAuth := false
	// var passAuth byte = 0x02
	//
	// check auth method
	// only password(0x02) supported
	// for i := 2; i < n; i++ {
	//	if buf[i] == passAuth {
	//		hasPassAuth = true
	//		break
	//	}
	// }
	//
	// if !hasPassAuth {
	//	_, _ = p.conn.Write([]byte{0x05, 0xff})
	//	log.Errorln(p.id, p.srcAddr(), "no supported auth method")
	//	return errors.New("no supported auth method")
	// }
	//
	// return p.passwordAuth()
}

func (p *Proxy) passwordAuth() error {
	buf := make([]byte, 32)

	// username/password required
	_, _ = p.conn.Write([]byte{0x05, 0x02})
	n, err := io.ReadAtLeast(p.conn, buf, 2)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	// check auth version
	if buf[0] != 0x01 {
		logrus.Errorln(p.id, p.srcAddr(), "unsupported auth version")
		return errors.New("unsupported auth version")
	}

	usernameLen := int(buf[1])
	p0 := 2
	p1 := p0 + usernameLen
	for n < p1 {
		var n1 int
		n1, err = p.conn.Read(buf[n:])
		if err != nil {
			logrus.Errorln(p.id, p.srcAddr(), err)
			return err
		}
		n += n1
	}
	user := string(buf[p0:p1])
	logrus.Infoln(p.id, p.srcAddr(), user)
	passwordLen := int(buf[p1])

	p3 := p1 + 1
	p4 := p3 + passwordLen

	for n < p4 {
		var n1 int
		n1, err = p.conn.Read(buf[n:])
		if err != nil {
			logrus.Errorln(p.id, p.srcAddr(), err)
			return err
		}
		n += n1
	}

	password := buf[p3:p4]
	// TODO: VerifyUser
	logrus.Debugln(p.id, p.srcAddr(), user, string(password))

	_, _ = p.conn.Write([]byte{0x01, 0x01})
	logrus.Errorln(p.id, p.srcAddr(), "access denied")
	return errors.New("access denied")
}

func (p *Proxy) processRequest(tcpIn chan<- *constant.TCPContext) error {
	buf := make([]byte, 258)

	// read header
	n, err := io.ReadAtLeast(p.conn, buf, 10)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}

	if buf[0] != Version {
		logrus.Errorln(p.id, p.srcAddr(), "error version", buf[0])
		return errors.New("error version")
	}

	hlen := 0   // target address length
	host := ""  // target address
	msglen := 0 // header length

	switch buf[3] {
	case constant.ATypeIPv4:
		hlen = 4
	case constant.ATypeDomainName:
		hlen = int(buf[4]) + 1
	case constant.ATypeIPv6:
		hlen = 16
	}

	msglen = 6 + hlen

	if n < msglen {
		// read remains header
		_, err = io.ReadFull(p.conn, buf[n:msglen])
		if err != nil {
			logrus.Errorln(p.id, p.srcAddr(), err)
			return err
		}
	}

	// get target address
	addr := buf[4 : 4+hlen]
	if buf[3] == constant.ATypeDomainName {
		host = string(addr[1:])
	} else {
		host = net.IP(addr).String()
	}

	// get target port
	port := binary.BigEndian.Uint16(buf[msglen-2 : msglen])

	// target address
	targetAddr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	// command support connect
	switch buf[1] {
	case CmdUdp:
		return p.handleUdpCmd()
	case CmdConnect, CmdBind:
		return p.handleConnectCmd(targetAddr, tcpIn)
	default:
		return errors.New("command not supported")
	}
}

func (p *Proxy) handleConnectCmd(targetAddr string, tcpIn chan<- *constant.TCPContext) error {
	client, err := constant.UnmarshalIP(p.srcAddr())
	if err != nil {
		return err
	}
	source, err := constant.UnmarshalIP(p.conn.LocalAddr().String())
	if err != nil {
		return err
	}
	target, err := constant.UnmarshalIP(targetAddr)
	if err != nil {
		return err
	}
	tcpIn <- &constant.TCPContext{
		SrcConn: p.conn,
		Metadata: &constant.Metadata{
			ID:      p.id,
			NetWork: constant.TCP,
			Type:    constant.SOCKS5,
			Client:  client,
			Source:  source,
			Target:  target,
		},
		PreFn: func() {
			_, err = p.conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01})
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

func (p *Proxy) handleUdpCmd() error {
	host, port, err := net.SplitHostPort(p.Udp)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	_port, err := strconv.Atoi(port)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	udpAddr, err := net.ResolveIPAddr("ip", host)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	hostByte := udpAddr.IP.To4()
	portByte := make([]byte, 2)
	binary.BigEndian.PutUint16(portByte, uint16(_port))
	buf := append([]byte{Version, 0x00, 0x00, 0x01}, hostByte...)
	buf = append(buf, portByte...)
	_, err = p.conn.Write(buf)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), "write response error", err)
		return err
	}

	forward := func(src net.Conn) {
		defer func(src net.Conn) {
			_ = src.Close()
		}(src)
		for {
			_, err = io.ReadFull(src, make([]byte, 100))
			if err != nil {
				if err != io.EOF {
					logrus.Errorln(p.id, p.srcAddr(), err)
				}
				break
			}
		}
	}

	forward(p.conn)
	return nil
}
