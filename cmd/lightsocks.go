package main

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/kardianos/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	info "github.com/xmapst/lightsocks"
	"github.com/xmapst/lightsocks/internal/api"
	"github.com/xmapst/lightsocks/internal/config"
	"github.com/xmapst/lightsocks/internal/lightsocks"
	"github.com/xmapst/lightsocks/internal/log"
	"github.com/xmapst/lightsocks/internal/mixed"
	N "github.com/xmapst/lightsocks/internal/net"
	"github.com/xmapst/lightsocks/internal/tunnel"
	"github.com/xmapst/lightsocks/internal/udp"
)

var (
	configPath string
	cmd        = &cobra.Command{
		Use:               os.Args[0],
		Short:             "Support socks4, socks4a, socks5, socks5h, http proxy all in one,",
		DisableAutoGenTag: true,
		Run:               run,
	}
)

func init() {
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&log.ConsoleFormat{})
	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "config file path")
}

func main() {
	cobra.CheckErr(cmd.Execute())
}

func run(_ *cobra.Command, _ []string) {
	info.PrintHeadInfo()
	svc, err := service.New(&program{}, &service.Config{
		Name:        "lightsocks",
		DisplayName: "lightsocks",
		Description: "Support socks4, socks4a, socks5, socks5h, http proxy all in one",
	})
	if err != nil {
		logrus.Fatalln(err)
	}
	err = svc.Run()
	if err != nil {
		logrus.Fatalln(err)
	}
}

type program struct {
	server *N.Listener
}

func (p *program) Start(service.Service) error {
	// load conf
	err := config.Load(configPath)
	if err != nil {
		logrus.Fatalln(err)
	}
	tunnel.Start(config.App.Outbound)
	api.Server(config.App.Dashboard)
	p.server = N.NewServer(config.App.Inbound.Host, config.App.Inbound.Port)
	var handler N.IConnHandler
	var tcpIn = tunnel.TCPIn.In
	if config.RunMode == config.ServerMode {
		handler = &lightsocks.Server{
			Config: config.App.Inbound,
			TcpIn:  tcpIn,
		}
	} else {
		udpServer, err := udp.New(p.server.RawAddress())
		if err != nil {
			logrus.Errorln(err)
			return err
		}
		// udp
		go func() {
			logrus.Infoln("UDP Server Listening At:", udpServer.LocalAddr())
			udpServer.ListenAndServe()
		}()
		handler = &mixed.Server{
			Config: config.App.Inbound,
			TcpIn:  tcpIn,
			Udp:    udpServer.LocalAddr(),
		}
		p.println()
	}
	go func() {
		err = p.server.ListenAndServe(handler)
		if err != nil {
			logrus.Fatalln(err)
		}
	}()
	return err
}

func (p *program) Stop(service.Service) error {
	logrus.Infoln("shutdown server")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if p.server != nil {
		_ = p.server.Shutdown(ctx)
	}
	return nil
}

func (p *program) println() {
	if config.App.Inbound.Host == "" ||
		config.App.Inbound.Host == "0.0.0.0" ||
		config.App.Inbound.Host == "[::]" {
		addrs, _ := net.InterfaceAddrs()
		for _, value := range addrs {
			if ipnet, ok := value.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					logrus.Infof(
						"http://%s:%d socks4://%s:%d socks5://%s:%d",
						ipnet.IP.String(), config.App.Inbound.Port,
						ipnet.IP.String(), config.App.Inbound.Port,
						ipnet.IP.String(), config.App.Inbound.Port,
					)
				} else if ipnet.IP.To16() != nil {
					logrus.Infof(
						"http://[%s]:%d socks4://[%s]:%d socks5://[%s]:%d",
						ipnet.IP.String(), config.App.Inbound.Port,
						ipnet.IP.String(), config.App.Inbound.Port,
						ipnet.IP.String(), config.App.Inbound.Port,
					)
				}
			}
		}
	} else {
		logrus.Infof(
			"http://%s:%d socks4://%s:%d socks5://%s:%d",
			config.App.Inbound.Host, config.App.Inbound.Port,
			config.App.Inbound.Host, config.App.Inbound.Port,
			config.App.Inbound.Host, config.App.Inbound.Port,
		)
	}
}
