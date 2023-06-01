package http

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/lightsocks/internal/constant"
	N "github.com/xmapst/lightsocks/internal/net"
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

func (p *Proxy) New(wg *sync.WaitGroup, conf *constant.Server, uuid uuid.UUID, conn net.Conn) error {
	p.wg = wg
	p.id = uuid
	p.conn = conn
	p.server = conf
	return nil
}

func (p *Proxy) Handle(tcpIn chan<- *constant.TCPContext) error {
	lines, err := p.readString("\r\n")
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	if len(lines) < 2 {
		logrus.Errorln(p.id, p.srcAddr(), "request line error")
		return errors.New("request line error")
	}
	err = p.handshake(lines)
	if err != nil {
		logrus.Errorln(p.id, p.srcAddr(), err)
		return err
	}
	return p.processRequest(lines, tcpIn)
}

func (p *Proxy) handshake(lines []string) (err error) {
	var user, pass string
	for _, line := range lines {
		// get username/password
		if strings.HasPrefix(line, ProxyAuthorization) {
			line = strings.TrimPrefix(line, ProxyAuthorization)
			bs, err := base64.StdEncoding.DecodeString(line)
			if err != nil {
				logrus.Errorln(p.id, p.srcAddr(), err)
				continue
			}
			if bs == nil {
				continue
			}
			_auth := bytes.Split(bs, []byte(":"))
			if len(_auth) < 2 {
				continue
			}
			user, pass = string(_auth[0]), string(bytes.Join(_auth[1:], []byte(":")))
		}
	}

	// TODO: VerifyUser
	logrus.Debugln(p.id, p.srcAddr(), user, pass)
	return
}

func (p *Proxy) processRequest(lines []string, tcpIn chan<- *constant.TCPContext) error {
	requestLine := strings.Split(lines[0], " ")
	if len(requestLine) < 3 {
		logrus.Errorln(p.id, p.srcAddr(), "request line error")
		return N.NotFoundResponse().Write(p.conn)
	}
	method := requestLine[0]
	requestTarget := requestLine[1]
	version := requestLine[2]
	var err error
	if method == HTTPCONNECT {
		shp := strings.Split(requestTarget, ":")
		addr := shp[0]
		port, _ := strconv.Atoi(shp[1])
		err = p.handleHTTPConnectMethod(addr, uint16(port), tcpIn)
	} else {
		si := strings.Index(requestTarget, "//")
		restUrl := requestTarget[si+2:]
		if restUrl == "" {
			return N.NotFoundResponse().Write(p.conn)
		}
		port := 80
		ei := strings.Index(restUrl, "/")
		url := "/"
		hostPort := restUrl
		if ei != -1 {
			hostPort = restUrl[:ei]
			url = restUrl[ei:]
		}
		as := strings.Split(hostPort, ":")
		addr := as[0]
		if len(as) == 2 {
			port, _ = strconv.Atoi(as[1])
		}
		var header string
		for _, line := range lines[1:] {
			if strings.HasPrefix(line, ProxyAuthorization) {
				continue
			}
			if strings.HasPrefix(line, "Proxy-") {
				line = strings.TrimPrefix(line, "Proxy-")
			}
			header += fmt.Sprintf("%s\r\n", line)
		}
		newline := method + " " + url + " " + version + "\r\n" + header
		err = p.handleHTTPProxy(addr, uint16(port), newline, tcpIn)
	}
	return err
}

func (p *Proxy) httpWriteProxyHeader() {
	_, err := p.conn.Write([]byte("HTTP/1.1 200 OK Connection Established\r\n"))
	if err != nil {
		logrus.Warnln(p.id, p.srcAddr(), err)
		return
	}

	_, err = p.conn.Write([]byte(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123))))
	if err != nil {
		logrus.Warnln(p.id, p.srcAddr(), err)
		return
	}
	_, err = p.conn.Write([]byte("Transfer-Encoding: chunked\r\n"))
	if err != nil {
		logrus.Warnln(p.id, p.srcAddr(), err)
		return
	}
	_, err = p.conn.Write([]byte("\r\n"))
	if err != nil {
		logrus.Warnln(p.id, p.srcAddr(), err)
		return
	}
}

func (p *Proxy) handleHTTPConnectMethod(addr string, port uint16, tcpIn chan<- *constant.TCPContext) error {
	targetAddr := fmt.Sprintf("%s:%d", addr, port)
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
			Type:    constant.HTTPS,
			Client:  client,
			Source:  source,
			Target:  target,
		},
		PreFn: p.httpWriteProxyHeader,
		PostFn: func() {
			p.wg.Done()
		},
	}
	return nil
}

// Subsequent request lines are full paths, some servers may have problems
func (p *Proxy) handleHTTPProxy(addr string, port uint16, line string, tcpIn chan<- *constant.TCPContext) error {
	targetAddr := fmt.Sprintf("%s:%d", addr, port)
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
			Type:    constant.HTTP,
			Client:  client,
			Source:  source,
			Target:  target,
		},
		Line: line,
		PostFn: func() {
			p.wg.Done()
		},
	}
	return nil
}

func (p *Proxy) readString(delim string) ([]string, error) {
	var buf = make([]byte, 4096)
	_, err := io.ReadAtLeast(p.conn, buf, 1)
	if err != nil && err != io.EOF {
		logrus.Errorln(p.id, p.srcAddr(), err.Error())
		return nil, err
	}
	return strings.Split(string(buf), delim), nil
}
