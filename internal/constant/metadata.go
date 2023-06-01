package constant

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
)

type NetWork int

func (n NetWork) String() string {
	switch n {
	case TCP:
		return "tcp"
	case UDP:
		return "udp"
	default:
		return "unknown"
	}
}

func UnmarshalNetWork(s string) NetWork {
	switch strings.ToLower(s) {
	case "tcp":
		return TCP
	case "udp":
		return UDP
	default:
		return Unknown
	}
}

func (n NetWork) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.String())
}

type Type int

func (t Type) String() string {
	switch t {
	case HTTP:
		return "HTTP"
	case HTTPS:
		return "HTTPS"
	case SOCKS4:
		return "Socks4"
	case SOCKS5:
		return "Socks5"
	default:
		return "Unknown"
	}
}

func UnmarshalType(s string) Type {
	switch strings.ToLower(s) {
	case "http":
		return HTTP
	case "https":
		return HTTPS
	case "socks4":
		return SOCKS4
	case "socks5":
		return SOCKS5
	default:
		return Unknown
	}
}

func (t Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

type Metadata struct {
	ID      uuid.UUID `json:"-"`
	NetWork NetWork   `json:"Network"`
	Type    Type      `json:"Type"`
	Client  *IP       `json:"Client"`
	Source  *IP       `json:"Source"`
	Target  *IP       `json:"Target"`
}

func (m *Metadata) String() string {
	return fmt.Sprintf("%s#%s#%s#%s#%s", m.NetWork, m.Type, m.Client, m.Source, m.Target)
}

func UnmarshalMetadata(s string) (*Metadata, error) {
	slice := strings.Split(s, "#")
	if len(slice) != 5 {
		return nil, errors.New("illegal")
	}
	client, err := UnmarshalIP(slice[2])
	if err != nil {
		return nil, err
	}
	source, err := UnmarshalIP(slice[3])
	if err != nil {
		return nil, err
	}
	target, err := UnmarshalIP(slice[4])
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	return &Metadata{
		ID:      id,
		NetWork: UnmarshalNetWork(slice[0]),
		Type:    UnmarshalType(slice[1]),
		Client:  client,
		Source:  source,
		Target:  target,
	}, nil
}

type IP struct {
	Addr string `json:"Addr"`
	Port int64  `json:"Port"`
}

func (i IP) String() string {
	return net.JoinHostPort(i.Addr, strconv.FormatInt(i.Port, 10))
}

func (i IP) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

func UnmarshalIP(s string) (*IP, error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return nil, err
	}
	_port, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return nil, err
	}
	return &IP{
		Addr: host,
		Port: _port,
	}, nil
}
