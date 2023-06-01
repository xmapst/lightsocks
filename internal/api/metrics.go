package api

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	info "github.com/xmapst/lightsocks"
	"github.com/xmapst/lightsocks/internal/statistic"
)

var (
	// Global/instance level metrics
	connectionsGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(info.Name),
		Name:      "connections",
		Help:      "Number of current connections.",
	})
	totalDownloadGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(info.Name),
		Name:      "download_bytes",
		Help:      "Total data downloaded in bytes.",
	})
	totalUploadGauge = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: strings.ToLower(info.Name),
		Name:      "upload_bytes",
		Help:      "Total data uploaded in bytes.",
	})

	// Conenction level metrics
	connectionDownloadGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(info.Name),
			Subsystem: "connection",
			Name:      "download_bytes",
			Help:      "Total data uploaded in bytes per connection.",
		},
		[]string{"id"},
	)
	connectionUploadGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: strings.ToLower(info.Name),
			Subsystem: "connection",
			Name:      "upload_bytes",
			Help:      "Total data uploaded in bytes per connection.",
		},
		[]string{"id"},
	)
)

func collectMetrics() {
	// Clear any existing metrics since we refetch them all
	connectionDownloadGauges.Reset()
	connectionUploadGauges.Reset()
	snapshot := statistic.DefaultManager.Snapshot()

	connectionsGauge.Set(float64(len(snapshot.Connections)))
	totalDownloadGauge.Set(float64(snapshot.DownloadTotal))
	totalUploadGauge.Set(float64(snapshot.UploadTotal))

	for _, connection := range snapshot.Connections {
		client := connection.MetadataX().Client.String()
		connectionDownloadGauges.WithLabelValues(client).Set(float64(connection.DownloadTotalX()))
		connectionUploadGauges.WithLabelValues(client).Set(float64(connection.UploadTotalX()))
	}
}

func init() {
	prometheus.MustRegister(connectionsGauge)
	prometheus.MustRegister(totalDownloadGauge)
	prometheus.MustRegister(totalUploadGauge)
	prometheus.MustRegister(connectionDownloadGauges)
	prometheus.MustRegister(connectionUploadGauges)
}

func collectMetricsLoop() {
	ticker := time.NewTicker(600 * time.Millisecond)
	for range ticker.C {
		collectMetrics()
	}
}
