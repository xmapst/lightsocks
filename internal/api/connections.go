package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/xmapst/lightsocks/internal/statistic"
)

func getConnections(c *gin.Context) {
	if !websocket.IsWebSocketUpgrade(c.Request) {
		snapshot := statistic.DefaultManager.Snapshot()
		c.SecureJSON(http.StatusOK, snapshot)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.SecureJSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	intervalStr := c.DefaultQuery("interval", "1000")
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		c.SecureJSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	buf := &bytes.Buffer{}
	sendSnapshot := func() error {
		buf.Reset()
		snapshot := statistic.DefaultManager.Snapshot()
		if err = json.NewEncoder(buf).Encode(snapshot); err != nil {
			return err
		}

		return conn.WriteMessage(websocket.TextMessage, buf.Bytes())
	}

	if err = sendSnapshot(); err != nil {
		c.SecureJSON(http.StatusBadRequest, ErrBadRequest)
		return
	}

	tick := time.NewTicker(time.Millisecond * time.Duration(interval))
	defer tick.Stop()
	for range tick.C {
		if err = sendSnapshot(); err != nil {
			break
		}
	}
}

func closeConnection(c *gin.Context) {
	id := c.Param("id")
	snapshot := statistic.DefaultManager.Snapshot()
	for _, conn := range snapshot.Connections {
		if id == conn.ID() {
			_ = conn.Close()
			break
		}
	}
	c.SecureJSON(http.StatusOK, gin.H{
		"Code": http.StatusOK,
		"Msg":  "Closed",
	})
}

func closeAllConnections(c *gin.Context) {
	snapshot := statistic.DefaultManager.Snapshot()
	for _, conn := range snapshot.Connections {
		_ = conn.Close()
	}
	c.SecureJSON(http.StatusOK, gin.H{
		"Code": http.StatusOK,
		"Msg":  "Closed",
	})
}
