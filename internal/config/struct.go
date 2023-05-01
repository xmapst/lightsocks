package config

import (
	"github.com/xmapst/lightsocks/internal/constant"
	"github.com/xmapst/lightsocks/internal/dns"
)

var (
	defaultNameServers = []dns.NameServer{
		{
			Addr: "8.8.8.8.",
		},
		{
			Addr: "1.1.1.1",
		},
		{
			Addr: "233.5.5.5",
		},
		{
			Addr: "119.29.29.29",
		},
	}
)

const (
	DirectMode = "Direct"
	ClientMode = "Client"
	ServerMode = "Server"
)

type Config struct {
	RunMode   string           `yaml:""` // 模式
	Inbound   *constant.Server `yaml:""` // 服务端及客户端监听的本地端口
	Outbound  *constant.Server `yaml:""` // 远端服务器地址
	Dashboard *constant.Server `yaml:""` // Dashboard
	DNS       DNS              `yaml:""` // DNS配置
	Log       Log              `yaml:""` // 日志输出
}

type DNS struct {
	NameServers []string          `yaml:""`
	Hosts       map[string]string `yaml:""`
}

type Log struct {
	Filename   string `yaml:""`
	Level      string `yaml:",default=info"`
	MaxBackups int    `yaml:",default=7"`
	MaxSize    int    `yaml:",default=500"`
	MaxAge     int    `yaml:",default=28"`
	Compress   bool   `yaml:",default=true"`
}
