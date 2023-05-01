package mixed

import (
	"github.com/gofrs/uuid"
	"github.com/xmapst/lightsocks/internal/constant"
	"github.com/xmapst/lightsocks/internal/http"
	"github.com/xmapst/lightsocks/internal/socks4"
	"github.com/xmapst/lightsocks/internal/socks5"
	"net"
	"sync"
)

type Proxy interface {
	New(wg *sync.WaitGroup, conf *constant.Server, uuid uuid.UUID, conn net.Conn) error
	Handle(tcpIn chan<- *constant.TCPContext) error
}

func (s *Server) socks4() Proxy {
	return &socks4.Proxy{}
}

func (s *Server) socks5(udpAddr string) Proxy {
	return &socks5.Proxy{Udp: udpAddr}
}

func (s *Server) http() Proxy {
	return &http.Proxy{}
}
