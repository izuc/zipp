package remotemetrics

import (
	"time"

	"go.uber.org/atomic"

	"github.com/izuc/zipp/packages/app/remotemetrics"
)

var isMeshTimeSynced atomic.Bool

func checkSynced() {
	oldMeshTimeSynced := isMeshTimeSynced.Load()
	tts := deps.Protocol.Engine().IsSynced()
	if oldMeshTimeSynced != tts {
		var myID string
		if deps.Local != nil {
			myID = deps.Local.ID().String()
		}
		syncStatusChangedEvent := &remotemetrics.MeshTimeSyncChangedEvent{
			Type:           "sync",
			NodeID:         myID,
			MetricsLevel:   Parameters.MetricsLevel,
			Time:           time.Now(),
			ATT:            deps.Protocol.Engine().Clock.Accepted().Time(),
			RATT:           deps.Protocol.Engine().Clock.Accepted().RelativeTime(),
			CTT:            deps.Protocol.Engine().Clock.Confirmed().Time(),
			RCTT:           deps.Protocol.Engine().Clock.Confirmed().RelativeTime(),
			CurrentStatus:  tts,
			PreviousStatus: oldMeshTimeSynced,
		}
		remotemetrics.Events.MeshTimeSyncChanged.Trigger(syncStatusChangedEvent)
	}
}

func sendSyncStatusChangedEvent(syncUpdate *remotemetrics.MeshTimeSyncChangedEvent) {
	_ = deps.RemoteLogger.Send(syncUpdate)
}
