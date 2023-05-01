package constant

import (
	"crypto/tls"
	"github.com/sirupsen/logrus"
	"time"
)

type Server struct {
	// 通用配置
	Host    string        `yaml:""` // 地址
	Port    int64         `yaml:""` // 端口
	Token   string        `yaml:""` // 加密key
	TLS     *TLS          `yaml:""` // 证书
	Timeout time.Duration `yaml:""` // 连接超时时间

	// 出口特殊配置
	Interface   string `yaml:""` // 指定出口网卡
	RoutingMark int    `yaml:""` // linux 下可指定fwmark

	// 证书
	TLSConf *tls.Config
}

func (s *Server) Enable() bool {
	return s.Port != 0
}

func (s *Server) LoadTLS() {
	if s.TLS == nil {
		s.TLS = new(TLS)
	}

	if s.TLS.ServerName == "" {
		s.TLSConf.InsecureSkipVerify = true
	} else {
		s.TLSConf.ServerName = s.TLS.ServerName
		s.TLSConf.InsecureSkipVerify = false
	}

	if !s.TLS.Enable || s.TLS.Key == "" || s.TLS.Cert == "" {
		return
	}

	cer, err := tls.LoadX509KeyPair(s.TLS.Cert, s.TLS.Key)
	if err != nil {
		logrus.Fatalln(err)
	}

	s.TLSConf.InsecureSkipVerify = false
	s.TLSConf.Certificates = []tls.Certificate{cer}
	return
}

type TLS struct {
	Enable     bool   `yaml:""`
	ServerName string `yaml:""`
	Key        string `yaml:""`
	Cert       string `yaml:""`
}
