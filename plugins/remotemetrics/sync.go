package remotemetrics

import (
	"go.uber.org/atomic"

	"github.com/izuc/zipp/packages/node/clock"

	"github.com/izuc/zipp/packages/app/remotemetrics"
)

var isMeshTimeSynced atomic.Bool

func checkSynced() {
	oldMeshTimeSynced := isMeshTimeSynced.Load()
	tts := deps.Mesh.TimeManager.Synced()
	if oldMeshTimeSynced != tts {
		var myID string
		if deps.Local != nil {
			myID = deps.Local.ID().String()
		}
		syncStatusChangedEvent := &remotemetrics.MeshTimeSyncChangedEvent{
			Type:           "sync",
			NodeID:         myID,
			MetricsLevel:   Parameters.MetricsLevel,
			Time:           clock.SyncedTime(),
			ATT:            deps.Mesh.TimeManager.ATT(),
			RATT:           deps.Mesh.TimeManager.RATT(),
			CTT:            deps.Mesh.TimeManager.CTT(),
			RCTT:           deps.Mesh.TimeManager.RCTT(),
			CurrentStatus:  tts,
			PreviousStatus: oldMeshTimeSynced,
		}
		remotemetrics.Events.MeshTimeSyncChanged.Trigger(syncStatusChangedEvent)
	}
}

func sendSyncStatusChangedEvent(syncUpdate *remotemetrics.MeshTimeSyncChangedEvent) {
	_ = deps.RemoteLogger.Send(syncUpdate)
}
