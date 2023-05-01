package statistic

import (
	"github.com/gofrs/uuid"
	"github.com/xmapst/lightsocks/internal/constant"
	"go.uber.org/atomic"
	"net"
	"time"
)

type tracker interface {
	ID() string
	Close() error
}

type trackerInfo struct {
	UUID          uuid.UUID          `json:"ID"`
	Metadata      *constant.Metadata `json:"Metadata"`
	UploadTotal   *atomic.Int64      `json:"Upload"`
	DownloadTotal *atomic.Int64      `json:"Download"`
	Start         time.Time          `json:"Start"`
}

type TcpTracker struct {
	net.Conn `json:"-"`
	*trackerInfo
	manager *Manager
}

func (tt *TcpTracker) ID() string {
	return tt.UUID.String()
}

func (tt *TcpTracker) Read(b []byte) (int, error) {
	n, err := tt.Conn.Read(b)
	download := int64(n)
	tt.manager.PushDownloaded(download)
	tt.DownloadTotal.Add(download)
	return n, err
}

func (tt *TcpTracker) Write(b []byte) (int, error) {
	n, err := tt.Conn.Write(b)
	upload := int64(n)
	tt.manager.PushUploaded(upload)
	tt.UploadTotal.Add(upload)
	return n, err
}

func (tt *TcpTracker) Close() error {
	tt.manager.Leave(tt)
	return tt.Conn.Close()
}

func NewTCPTracker(conn net.Conn, metadata *constant.Metadata) *TcpTracker {
	t := &TcpTracker{
		Conn:    conn,
		manager: DefaultManager,
		trackerInfo: &trackerInfo{
			UUID:          metadata.ID,
			Start:         time.Now(),
			Metadata:      metadata,
			UploadTotal:   atomic.NewInt64(0),
			DownloadTotal: atomic.NewInt64(0),
		},
	}
	DefaultManager.Join(t)
	return t
}
