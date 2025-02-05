package remotemetrics

import (
	"time"

	"github.com/izuc/zipp/packages/core/mesh_old"

	"github.com/izuc/zipp/packages/app/remotemetrics"
)

func obtainSchedulerStats(timestamp time.Time) {
	scheduler := deps.Mesh.Scheduler
	queueMap, aManaNormalizedMap := prepQueueMaps(scheduler)

	var myID string
	if deps.Local != nil {
		myID = deps.Local.Identity.ID().String()
	}
	record := remotemetrics.SchedulerMetrics{
		Type:                         "schedulerSample",
		NodeID:                       myID,
		Synced:                       deps.Mesh.Synced(),
		MetricsLevel:                 Parameters.MetricsLevel,
		BufferSize:                   uint32(scheduler.BufferSize()),
		BufferLength:                 uint32(scheduler.TotalBlocksCount()),
		ReadyBlocksInBuffer:          uint32(scheduler.ReadyBlocksCount()),
		QueueLengthPerNode:           queueMap,
		AManaNormalizedLengthPerNode: aManaNormalizedMap,
		Timestamp:                    timestamp,
	}

	_ = deps.RemoteLogger.Send(record)
}

func prepQueueMaps(s *mesh_old.Scheduler) (queueMap map[string]uint32, aManaNormalizedMap map[string]float64) {
	queueSizes := s.NodeQueueSizes()
	queueMap = make(map[string]uint32, len(queueSizes))
	aManaNormalizedMap = make(map[string]float64, len(queueSizes))

	for id, size := range queueSizes {
		nodeID := id.String()
		aMana := s.GetManaFromCache(id)

		queueMap[nodeID] = uint32(size)
		aManaNormalizedMap[nodeID] = float64(size) / float64(aMana)
	}
	return
}
