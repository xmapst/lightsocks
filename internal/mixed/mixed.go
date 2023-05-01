package mixed

import (
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/lightsocks/internal/constant"
	N "github.com/xmapst/lightsocks/internal/net"
	"github.com/xmapst/lightsocks/internal/socks4"
	"github.com/xmapst/lightsocks/internal/socks5"
	"io"
	"net"
	"sync"
)

type Server struct {
	Config *constant.Server
	TcpIn  chan<- *constant.TCPContext
	Udp    string
}

func (s *Server) Handler(wg *sync.WaitGroup, conn net.Conn) {
	wg.Add(1)
	var err error
	defer func() {
		if err != nil {
			wg.Done()
			_ = conn.Close()
		}
	}()

	id, _ := uuid.NewV4()
	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		if err != io.EOF {
			logrus.Errorln(id, conn.RemoteAddr(), err)
		}
		return
	}
	var proxy Proxy
	switch head[0] {
	case socks4.Version:
		proxy = s.socks4()
	case socks5.Version:
		proxy = s.socks5(s.Udp)
	default:
		proxy = s.http()
	}
	err = proxy.New(wg, s.Config, id, bufConn)
	if err != nil {
		if err != io.EOF {
			logrus.Errorln(id, conn.RemoteAddr(), err)
		}
		return
	}
	err = proxy.Handle(s.TcpIn)
	if err != nil {
		if err != io.EOF {
			logrus.Errorln(id, conn.RemoteAddr(), err)
		}
		return
	}
}
