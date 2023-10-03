package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/izuc/zipp/plugins/banner"
	"github.com/izuc/zipp/plugins/metrics"
)

var (
	infoApp              *prometheus.GaugeVec
	meshTimeSyncStatus prometheus.Gauge
	nodeID               string
)

func registerInfoMetrics() {
	infoApp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "iota_info_app",
			Help: "Node software name and version.",
		},
		[]string{"name", "version", "nodeID"},
	)
	if deps.Local != nil {
		nodeID = deps.Local.ID().String()
	}
	infoApp.WithLabelValues(banner.AppName, banner.AppVersion, nodeID).Set(1)

	meshTimeSyncStatus = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "meshTimeSynced",
		Help: "Node sync status based on MeshTime.",
	})

	registry.MustRegister(infoApp)
	registry.MustRegister(meshTimeSyncStatus)

	addCollect(collectInfoMetrics)
}

func collectInfoMetrics() {
	meshTimeSyncStatus.Set(func() float64 {
		if metrics.MeshTimeSynced() {
			return 1
		}
		return 0
	}())
}
