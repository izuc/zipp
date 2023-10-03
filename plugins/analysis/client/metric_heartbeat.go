package client

import (
	"io"
	"runtime"
	"time"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/shirou/gopsutil/cpu"

	"github.com/izuc/zipp/packages/app/metrics"
	"github.com/izuc/zipp/plugins/analysis/packet"
	"github.com/izuc/zipp/plugins/banner"
)

func sendMetricHeartbeat(w io.Writer, hb *packet.MetricHeartbeat) {
	data, err := packet.NewMetricHeartbeatBlock(hb)
	if err != nil {
		log.Debugw("metric heartbeat block skipped", "err", err)
		return
	}

	if _, err = w.Write(data); err != nil {
		log.Debugw("Error while writing to connection", "Description", err)
	}
	// trigger AnalysisOutboundBytes event
	metrics.Events.AnalysisOutboundBytes.Trigger(&metrics.AnalysisOutboundBytesEvent{AmountBytes: uint64(len(data))})
}

func createMetricHeartbeat() *packet.MetricHeartbeat {
	// get own ID
	nodeID := make([]byte, len(identity.ID{}))
	if deps.Local != nil {
		nodeIDBytes, err := deps.Local.ID().Bytes()
		if err != nil {
			log.Error("Failed to get bytes from node ID: ", err)
			return nil // or handle the error as appropriate
		}
		copy(nodeID, nodeIDBytes)
	}

	return &packet.MetricHeartbeat{
		Version: banner.SimplifiedAppVersion,
		OwnID:   nodeID,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		NumCPU:  runtime.GOMAXPROCS(0),
		// TODO: replace this with only the CPU usage of the ZIPP process.
		CPUUsage: func() (p float64) {
			percent, err := cpu.Percent(time.Second, false)
			if err == nil {
				p = percent[0]
			}
			return
		}(),
		MemoryUsage: func() uint64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return m.Alloc
		}(),
	}
}
