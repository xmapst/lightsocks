package tunnel

import (
	"context"
	"net"
	"runtime"
	"time"

	"github.com/refraction-networking/utls"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/chanx"
	"github.com/xmapst/lightsocks/internal/config"
	"github.com/xmapst/lightsocks/internal/constant"
	"github.com/xmapst/lightsocks/internal/dialer"
	N "github.com/xmapst/lightsocks/internal/net"
	"github.com/xmapst/lightsocks/internal/statistic"
)

var (
	TCPIn         = chanx.NewUnboundedChan[*constant.TCPContext](10000)
	DefaultWorker = 4
)

func Start(server *constant.Server) {
	go process(server)
}

// processTCP starts a loop to handle tcp packet
func processTCP(server *constant.Server) {
	for conn := range TCPIn.Out {
		go handleTCPConn(conn, server)
	}
}

func process(server *constant.Server) {
	if num := runtime.GOMAXPROCS(0); num > DefaultWorker {
		DefaultWorker = num
	}
	DefaultWorker *= DefaultWorker
	for i := 0; i < DefaultWorker; i++ {
		go processTCP(server)
	}
}

func handleTCPConn(ctx *constant.TCPContext, server *constant.Server) {
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(ctx.SrcConn)

	// connect to the target
	var target = ctx.Metadata.Target
	if config.RunMode == config.ClientMode && server.Enable() {
		target = &constant.IP{
			Addr: server.Host,
			Port: server.Port,
		}
	}

	var destConn net.Conn
	var err error
	destConn, err = dialer.DialContext(
		context.Background(), "tcp", target.String(),
		dialer.WithTimeout(server.Timeout), dialer.WithInterface(server.Interface),
		dialer.WithRoutingMark(server.RoutingMark),
	)
	if err != nil {
		logrus.Errorln(ctx.Metadata.ID, "-->", ctx.Metadata.Client, "-->", ctx.Metadata.Source, "-->", ctx.Metadata.Target, err.Error())
		return
	}
	if server.TLS.Enable && config.RunMode == config.ClientMode {
		helloID := tls.ClientHelloID{}
		switch server.TLS.Fingerprint {
		case "firefox":
			helloID = tls.HelloFirefox_Auto
		case "chrome":
			helloID = tls.HelloChrome_Auto
		case "ios":
			helloID = tls.HelloIOS_Auto
		default:
			helloID = tls.HelloFirefox_Auto
		}
		tlsConn := tls.UClient(destConn, server.TLSConf, helloID)
		err = tlsConn.Handshake()
		if err != nil {
			logrus.Errorln(ctx.Metadata.ID, "-->", ctx.Metadata.Client, "-->", ctx.Metadata.Source, "-->", ctx.Metadata.Target, err.Error())
			return
		}
		destConn = tlsConn
	}
	// 连接管理
	destConn = statistic.NewTCPTracker(destConn, ctx.Metadata)
	defer func(destConn net.Conn) {
		_ = destConn.Close()
	}(destConn)

	// 激活4层会话保持
	tcpKeepAlive(destConn)

	// 发送http代理头信息
	err = sedHttpHeader(ctx, destConn)
	if err != nil {
		return
	}

	if ctx.PreFn != nil {
		// 通道开启前, 预处理, 例如:
		// 1. socks代理需要发送连接成功信息给客户端
		// 2. http代理需要发送代理头给客户端
		ctx.PreFn()
	}

	defer func() {
		// 通道完成后的处理
		if ctx.PostFn != nil {
			ctx.PostFn()
		}
	}()

	var src, dest, _type = ctx.SrcConn, destConn, constant.Direct
	if config.RunMode != config.DirectMode {
		_type = constant.Proxy
		if config.RunMode == config.ClientMode {
			src, dest = destConn, ctx.SrcConn
		}
	}
	relay := &N.Relay{
		Src:      src,
		Dest:     dest,
		Metadata: ctx.Metadata,
		Token:    []byte(config.Token),
	}
	relay.Start(_type)
}

func tcpKeepAlive(c net.Conn) {
	if tcp, ok := c.(*net.TCPConn); ok {
		_ = tcp.SetKeepAlive(true)
		_ = tcp.SetKeepAlivePeriod(30 * time.Second)
	}
}

func sedHttpHeader(ctx *constant.TCPContext, destConn net.Conn) (err error) {
	if config.RunMode == config.ClientMode {
		// 客户端模式需要提前写入被代理地址信息到远端服务器
		destSecConn := &N.SecureTCPConn{ReadWriteCloser: destConn}
		_, err = destSecConn.EncodeWrite([]byte(config.Token), []byte(ctx.Metadata.String()))
		if err != nil {
			logrus.Errorln(ctx.Metadata.ID, "-->", ctx.Metadata.Client, "-->", ctx.Metadata.Source, "-->", ctx.Metadata.Target, err)
			return
		}
	}
	if ctx.Line != "" {
		switch config.RunMode {
		case config.ClientMode:
			// 客户端模式使用加密方式写入远端服务器
			destSecConn := &N.SecureTCPConn{ReadWriteCloser: destConn}
			// redirect http proxy
			_, err = destSecConn.EncodeWrite([]byte(config.Token), []byte(ctx.Line))
			if err != nil {
				logrus.Errorln(ctx.Metadata.ID, "-->", ctx.Metadata.Client, "-->", ctx.Metadata.Source, "-->", ctx.Metadata.Target, err)
				return
			}
		default:
			_, err = destConn.Write([]byte(ctx.Line))
			if err != nil {
				logrus.Errorln(ctx.Metadata.ID, "-->", ctx.Metadata.Client, "-->", ctx.Metadata.Source, "-->", ctx.Metadata.Target, err)
				return
			}
		}
	}
	return
}
