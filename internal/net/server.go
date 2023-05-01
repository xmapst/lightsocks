package net

import (
	"bytes"
	"context"
	"fmt"
	"github.com/pires/go-proxyproto"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type IConnHandler interface {
	Handler(wg *sync.WaitGroup, conn net.Conn)
}

type Listener struct {
	tcp     net.Listener
	wg      *sync.WaitGroup
	Addr    string
	Port    int64
	handler IConnHandler
}

func (l *Listener) RawAddress() string {
	return fmt.Sprintf("%s:%d", l.Addr, l.Port)
}

func (l *Listener) Address() string {
	return l.tcp.Addr().String()
}

func (l *Listener) close() {
	if l.tcp != nil {
		_ = l.tcp.Close()
		return
	}
	return
}

func (l *Listener) State() bool {
	return l.tcp != nil
}

func (l *Listener) Shutdown(ctx context.Context) error {
	l.close()
	c := make(chan struct{})
	go func() {
		defer close(c)
		l.wg.Wait()
	}()
	defer func() {
		logrus.Infoln("server closed")
		l.tcp = nil
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c:
		return nil
	}
}

func NewServer(addr string, port int64) *Listener {
	return &Listener{
		wg:   new(sync.WaitGroup),
		Addr: addr,
		Port: port,
	}
}

func (l *Listener) ListenAndServe(handler IConnHandler) (err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", l.RawAddress())
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	l.tcp, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	logrus.Infoln("TCP Server Listening At:", l.tcp.Addr())
	ln := &proxyproto.Listener{Listener: l.tcp}
	for {
		var conn net.Conn
		conn, err = ln.Accept()
		if err != nil {
			continue
		}
		go handler.Handler(l.wg, conn)
	}
}

const NotFound = `<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
</html>
`

func NotFoundResponse() *http.Response {
	header := make(http.Header)
	header.Set("server", "nginx/1.22.0")
	header.Set("Content-Type", "text/html")
	header.Set("date", time.Now().Format(time.RFC1123))
	header.Set("Cache-Control", "no-cache, must-revalidate")
	header.Set("Connection", "keep-alive")
	header.Set("Expect", time.Now().Format(time.RFC1123))
	header.Set("Pragma", "no-cache")

	content := []byte(NotFound)
	res := &http.Response{
		Status:        "Not Found",
		StatusCode:    http.StatusNotFound,
		Proto:         "HTTP/3",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(content)),
		ContentLength: int64(len(content)),
	}
	return res
}
