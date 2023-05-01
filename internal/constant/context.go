package constant

import (
	"net"
)

const Unknown = 999

// Socks addr type
const (
	TCP NetWork = iota
	UDP
)

const (
	HTTP Type = iota
	HTTPS
	SOCKS4
	SOCKS5
)

const (
	Direct = iota
	Proxy
	Block
)

// TCPContext is used to store connection address
type TCPContext struct {
	SrcConn  net.Conn
	Metadata *Metadata
	Line     string // http proxy
	PreFn    func()
	PostFn   func()
}
