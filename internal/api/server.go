package api

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/refraction-networking/utls"
	"github.com/sirupsen/logrus"
	info "github.com/xmapst/lightsocks"
	"github.com/xmapst/lightsocks/internal/constant"
	"github.com/xmapst/lightsocks/internal/log"
	"github.com/xmapst/lightsocks/internal/statistic"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

func Server(server *constant.Server) {
	if !server.Enable() {
		return
	}
	router := gin.New()
	router.Use(
		cors.New(cors.Config{
			AllowAllOrigins: true,
			AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			AllowHeaders:    []string{"Content-Type", "Authorization"},
			MaxAge:          300,
		}),
		timeoutMiddleware(server.Timeout),
		gin.Recovery(),
	)
	pprof.Register(router)

	api := router.Group("/api", func(c *gin.Context) {
		if server.Token == "" {
			c.Next()
			return
		}
		// Browser websocket not support custom header
		if websocket.IsWebSocketUpgrade(c.Request) && c.Query("token") != "" {
			token := c.Query("token")
			if token != server.Token {
				c.SecureJSON(http.StatusUnauthorized, ErrUnauthorized)
				c.Abort()
				return
			}
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		bearer, token, found := strings.Cut(header, " ")

		hasInvalidHeader := bearer != "Bearer"
		hasInvalidSecret := !found || token != server.Token
		if hasInvalidHeader || hasInvalidSecret {
			c.SecureJSON(http.StatusUnauthorized, ErrUnauthorized)
			c.Abort()
			return
		}
	})
	{
		api.GET("/version", version)
		api.GET("/logs", getLogs)
		api.GET("/traffic", traffic)
		api.GET("/connections", getConnections)
		api.DELETE("/connections", closeAllConnections)
		api.DELETE("/connections/:id", closeConnection)
		api.GET("/dns/query", queryDNS)
	}
	// prometheus
	router.GET("/metrics", func(c *gin.Context) {
		h := promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
				ErrorLog:      logrus.StandardLogger(),  // 采集过程中如果出现错误，记录日志
				ErrorHandling: promhttp.ContinueOnError, // 采集过程中如果出现错误，继续采集其他数据，不会中断采集器的工作
				Timeout:       server.Timeout,           // 超时时间
			}),
		)
		h.ServeHTTP(c.Writer, c.Request)
	})
	go collectMetricsLoop()

	// dashboard静态页面
	router.Use(info.StaticFile("/"))

	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port))
	if err != nil {
		logrus.Errorln("dashboard listen error:", err)
		return
	}
	if server.TLS.Enable {
		ln = tls.NewListener(ln, server.TLSConf)
	}
	logrus.Infoln("dashboard listening At:", ln.Addr())
	logrus.Infoln()
	go func() {
		if err = http.Serve(ln, router); err != nil {
			logrus.Errorln("dashboard serve error:", err)
		}
	}()
}

func timeoutResponse(c *gin.Context) {
	c.SecureJSON(http.StatusGatewayTimeout, newError("Timeout"))
}

func timeoutMiddleware(duration time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(duration),
		timeout.WithHandler(func(c *gin.Context) {
			c.Next()
		}),
		timeout.WithResponse(timeoutResponse),
	)
}

func version(c *gin.Context) {
	c.SecureJSON(http.StatusOK, gin.H{
		"Name":      info.Name,
		"Version":   info.Version,
		"BuildTime": info.BuildTime,
		"GO": gin.H{
			"OS":      info.GoOs,
			"ARCH":    info.GoArch,
			"Version": info.GoVersion,
		},
		"Git": gin.H{
			"Url":    info.GitUrl,
			"Branch": info.GitBranch,
			"Commit": info.GitCommit,
		},
	})
}

type Traffic struct {
	Up   int64 `json:"Up"`
	Down int64 `json:"Down"`
}

func traffic(c *gin.Context) {
	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(c.Request) {
		var err error
		wsConn, err = upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.SecureJSON(http.StatusBadRequest, newError(err.Error()))
			return
		}
	}

	if wsConn == nil {
		c.Header("Content-Type", "application/json")
		c.Status(http.StatusOK)
	}

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	t := statistic.DefaultManager
	buf := &bytes.Buffer{}
	var err error
	for range tick.C {
		buf.Reset()
		up, down := t.Now()
		if err = json.NewEncoder(buf).Encode(Traffic{
			Up:   up,
			Down: down,
		}); err != nil {
			break
		}

		if wsConn == nil {
			_, err = c.Writer.Write(buf.Bytes())
			c.Writer.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
}

type Log struct {
	Type    string `json:"Type"`
	Payload string `json:"Payload"`
}

func getLogs(c *gin.Context) {
	levelText := c.DefaultQuery("level", "info")

	level, err := logrus.ParseLevel(levelText)
	if err != nil {
		level = logrus.InfoLevel
	}

	var wsConn *websocket.Conn
	if websocket.IsWebSocketUpgrade(c.Request) {
		wsConn, err = upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}
	}

	if wsConn == nil {
		c.Header("Content-Type", "application/json")
		c.Status(http.StatusOK)
	}

	ch := make(chan log.Event, 1024)
	sub := log.Subscribe()
	defer log.UnSubscribe(sub)
	buf := &bytes.Buffer{}

	go func() {
		for elm := range sub {
			_log := elm.(log.Event)
			select {
			case ch <- _log:
			default:
			}
		}
		close(ch)
	}()

	for _log := range ch {
		if _log.LogLevel > level {
			continue
		}
		buf.Reset()

		if err = json.NewEncoder(buf).Encode(Log{
			Type:    _log.Type(),
			Payload: _log.Payload,
		}); err != nil {
			break
		}

		if wsConn == nil {
			_, err = c.Writer.Write(buf.Bytes())
			c.Writer.(http.Flusher).Flush()
		} else {
			err = wsConn.WriteMessage(websocket.TextMessage, buf.Bytes())
		}

		if err != nil {
			break
		}
	}
}
