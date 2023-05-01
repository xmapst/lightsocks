package lightsocks

import (
	"crypto/tls"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/lightsocks/internal/constant"
	N "github.com/xmapst/lightsocks/internal/net"
	"github.com/xmapst/lightsocks/internal/protocol"
	"github.com/xmapst/lightsocks/internal/resolver"
	"io"
	"net"
	"sync"
)

type Server struct {
	Config *constant.Server
	TcpIn  chan<- *constant.TCPContext
}

func (s *Server) Handler(wg *sync.WaitGroup, conn net.Conn) {
	wg.Add(1)
	var err error
	defer func() {
		if err != nil {
			_ = N.NotFoundResponse().Write(conn)
			wg.Done()
			_ = conn.Close()
		}
	}()

	if s.Config.TLS.Enable {
		tlsConn := tls.Server(conn, s.Config.TLSConf)
		err = tlsConn.Handshake()
		if err != nil {
			logrus.Errorln(conn.RemoteAddr(), err)
			return
		}
		conn = tlsConn
	}

	srcConn := N.NewBufferedConn(conn)
	metadata, err := s.getHeader(srcConn)
	if err != nil {
		if err != io.EOF {
			logrus.Errorln(conn.RemoteAddr(), err)
		}
		return
	}
	err = s.checkHost(metadata.Target)
	if err != nil {
		if err != io.EOF {
			logrus.Errorln(conn.RemoteAddr(), err)
		}
		return
	}
	s.TcpIn <- &constant.TCPContext{
		SrcConn:  srcConn,
		Metadata: metadata,
		PostFn: func() {
			wg.Done()
		},
	}
}

func (s *Server) getHeader(conn net.Conn) (*constant.Metadata, error) {
	packet, err := protocol.ReadFull([]byte(s.Config.Token), conn)
	if err != nil {
		return nil, err
	}
	metadata, err := constant.UnmarshalMetadata(string(packet.Payload))
	if err != nil {
		return nil, err
	}
	source, err := constant.UnmarshalIP(conn.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	metadata.Source = source
	return metadata, nil
}

func (s *Server) checkHost(destAddr *constant.IP) error {
	_, err := resolver.ResolveIP(destAddr.Addr)
	return err
}
