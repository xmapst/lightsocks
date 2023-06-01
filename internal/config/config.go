package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/refraction-networking/utls"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/xmapst/lightsocks/internal/constant"
	"github.com/xmapst/lightsocks/internal/dns"
	"github.com/xmapst/lightsocks/internal/resolver"
	"github.com/xmapst/lightsocks/internal/trie"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	App       *Config
	RunMode   string
	Token     string
	logOutput *lumberjack.Logger
	v         = viper.NewWithOptions(viper.KeyDelimiter("::"))
)

func viperLoadConf() error {
	err := v.ReadInConfig()
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	var conf = &Config{
		RunMode: DirectMode,
		Inbound: &constant.Server{
			Timeout: 30 * time.Second,
			TLSConf: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		},
		Outbound: &constant.Server{
			Timeout: 30 * time.Second,
			TLSConf: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		},
		Dashboard: &constant.Server{
			Timeout: 30 * time.Second,
			TLSConf: &tls.Config{
				MinVersion: tls.VersionTLS13,
			},
		},
		Log: Log{
			Level:      "info",
			MaxBackups: 7,
			MaxSize:    500,
			MaxAge:     28,
			Compress:   true,
		},
	}
	err = v.Unmarshal(conf)
	if err != nil {
		logrus.Errorln(err)
		return err
	}
	if conf.Dashboard != nil {
		conf.Dashboard.LoadTLS()
	}
	conf.Inbound.LoadTLS()
	conf.Outbound.LoadTLS()

	if !conf.Outbound.Enable() && conf.RunMode != ServerMode {
		conf.RunMode = DirectMode
	}
	if !conf.Inbound.Enable() {
		return errors.New("inbound is not enable")
	}
	if conf.RunMode == ServerMode {
		Token = conf.Inbound.Token
	}
	if conf.RunMode == ClientMode {
		Token = conf.Outbound.Token
	}
	nameServers, err := conf.parseNameServer()
	if err != nil {
		return err
	}
	resolver.DefaultResolver = dns.NewResolver(nameServers)
	resolver.DefaultHosts, err = conf.parseHosts()
	if err != nil {
		return err
	}
	RunMode = conf.RunMode
	App = conf
	return nil
}

func Load(filepath string) error {
	v.SetConfigFile(filepath)
	v.SetConfigType("yaml")
	err := viperLoadConf()
	if err != nil {
		return err
	}
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		if !e.Has(fsnotify.Write) {
			return
		}
		err = viperLoadConf()
		if err != nil {
			logrus.Warnln(err.Error())
			return
		}
		err = App.load()
		if err != nil {
			logrus.Warnln(err.Error())
			return
		}
	})

	err = App.load()
	if err != nil {
		return err
	}
	_cron := cron.New()
	_, _ = _cron.AddFunc("@daily", func() {
		if logOutput != nil {
			_ = logOutput.Rotate()
		}
	})
	_cron.Start()
	return nil
}

func (c *Config) load() error {
	level, err := logrus.ParseLevel(c.Log.Level)
	if err != nil {
		logrus.Warnln(err.Error())
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)
	if c.Log.Filename != "" {
		logOutput = &lumberjack.Logger{
			Filename:   c.Log.Filename,
			MaxBackups: c.Log.MaxBackups,
			MaxSize:    c.Log.MaxSize,  // megabytes
			MaxAge:     c.Log.MaxAge,   // days
			Compress:   c.Log.Compress, // disabled by default
			LocalTime:  true,           // use local time zone
		}
		logrus.SetOutput(logOutput)
	} else {
		logOutput = nil
		logrus.SetOutput(os.Stdout)
	}
	return nil
}

func hostWithDefaultPort(host string, defPort string) (string, error) {
	if !strings.Contains(host, ":") {
		host += ":"
	}

	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		return "", err
	}

	if port == "" {
		port = defPort
	}

	return net.JoinHostPort(hostname, port), nil
}

func (c *Config) parseNameServer() ([]dns.NameServer, error) {
	var nameservers []dns.NameServer
	for idx, server := range c.DNS.NameServers {
		// parse without scheme .e.g 8.8.8.8:53
		if !strings.Contains(server, "://") {
			server = "udp://" + server
		}

		u, err := url.Parse(server)
		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}

		// parse with specific interface
		// .e.g 10.0.0.1#en0
		interfaceName := u.Fragment
		if interfaceName == "" {
			interfaceName = c.Outbound.Interface
		}

		var addr, dnsNetType string
		switch u.Scheme {
		case "udp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "" // UDP
		case "tcp":
			addr, err = hostWithDefaultPort(u.Host, "53")
			dnsNetType = "tcp" // TCP
		case "tls":
			addr, err = hostWithDefaultPort(u.Host, "853")
			dnsNetType = "tcp-tls" // DNS over TLS
		case "https":
			clearURL := url.URL{Scheme: "https", Host: u.Host, Path: u.Path}
			addr = clearURL.String()
			dnsNetType = "https" // DNS over HTTPS
		default:
			return nil, fmt.Errorf("DNS NameServer[%d] unsupport scheme: %s", idx, u.Scheme)
		}

		if err != nil {
			return nil, fmt.Errorf("DNS NameServer[%d] format error: %s", idx, err.Error())
		}
		nameservers = append(
			nameservers,
			dns.NameServer{
				Net:       dnsNetType,
				Addr:      addr,
				Interface: interfaceName,
			},
		)
	}
	if nameservers == nil {
		return defaultNameServers, nil
	}
	return nameservers, nil
}

func (c *Config) parseHosts() (*trie.DomainTrie, error) {
	tree := trie.New()
	// add default hosts
	if err := tree.Insert("localhost", net.IP{127, 0, 0, 1}); err != nil {
		logrus.Errorln("insert localhost to host error: %v", err)
	}
	for domain, ipStr := range c.DNS.Hosts {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf("%s is not a valid IP", ipStr)
		}
		_ = tree.Insert(domain, ip)
	}
	return tree, nil
}
