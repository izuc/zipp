package metrics

import (
	"go.uber.org/atomic"
)

var isMeshTimeSynced atomic.Bool

func measureSynced() {
	tts := deps.Mesh.TimeManager.Synced()
	isMeshTimeSynced.Store(tts)
}

// MeshTimeSynced returns if the node is synced based on mesh time.
func MeshTimeSynced() bool {
	return isMeshTimeSynced.Load()
}
