// Package remotemetrics is a plugin that enables log metrics too complex for Prometheus, but still interesting in terms of analysis and debugging.
// It is enabled by default.
// The destination can be set via logger.remotelog.serverAddress.
package remotemetrics

import (
	"context"
	"time"

	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/types"

	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/ledger/utxo"

	"github.com/izuc/zipp/packages/app/remotemetrics"
	"github.com/izuc/zipp/packages/node/shutdown"
	"github.com/izuc/zipp/plugins/remotelog"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/izuc/zipp.foundation/core/timeutil"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/conflictdag"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

const (
	syncUpdateTime           = 500 * time.Millisecond
	schedulerQueryUpdateTime = 5 * time.Second
)

const (
	// Debug defines the most verbose metrics collection level.
	Debug uint8 = iota
	// Info defines regular metrics collection level.
	Info
	// Important defines the level of collection of only most important metrics.
	Important
	// Critical defines the level of collection of only critical metrics.
	Critical
)

var (
	// Plugin is the plugin instance of the remote plugin instance.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In

	Local        *peer.Local
	Mesh       *mesh_old.Mesh
	RemoteLogger *remotelog.RemoteLoggerConn `optional:"true"`
	ClockPlugin  *node.Plugin                `name:"clock" optional:"true"`
}

func init() {
	Plugin = node.NewPlugin("RemoteLogMetrics", deps, node.Enabled, configure, run)
}

func configure(_ *node.Plugin) {
	// if remotelog plugin is disabled, then remotemetrics should not be started either
	if node.IsSkipped(remotelog.Plugin) {
		Plugin.LogInfof("%s is disabled; skipping %s\n", remotelog.Plugin.Name, Plugin.Name)
		return
	}
	measureInitialConflictCounts()
	configureSyncMetrics()
	configureConflictConfirmationMetrics()
	configureBlockFinalizedMetrics()
	configureBlockScheduledMetrics()
	configureMissingBlockMetrics()
	configureSchedulerQueryMetrics()
}

func run(_ *node.Plugin) {
	// if remotelog plugin is disabled, then remotemetrics should not be started either
	if node.IsSkipped(remotelog.Plugin) {
		return
	}
	// create a background worker that update the metrics every second
	if err := daemon.BackgroundWorker("Node State Logger Updater", func(ctx context.Context) {
		// Do not block until the Ticker is shutdown because we might want to start multiple Tickers and we can
		// safely ignore the last execution when shutting down.
		timeutil.NewTicker(func() { checkSynced() }, syncUpdateTime, ctx)
		timeutil.NewTicker(func() { remotemetrics.Events.SchedulerQuery.Trigger(&remotemetrics.SchedulerQueryEvent{time.Now()}) }, schedulerQueryUpdateTime, ctx)

		// Wait before terminating so we get correct log blocks from the daemon regarding the shutdown order.
		<-ctx.Done()
	}, shutdown.PriorityRemoteLog); err != nil {
		Plugin.Panicf("Failed to start as daemon: %s", err)
	}
}

func configureSyncMetrics() {
	if Parameters.MetricsLevel > Info {
		return
	}
	remotemetrics.Events.MeshTimeSyncChanged.Attach(event.NewClosure(func(event *remotemetrics.MeshTimeSyncChangedEvent) {
		isMeshTimeSynced.Store(event.CurrentStatus)
	}))
	remotemetrics.Events.MeshTimeSyncChanged.Attach(event.NewClosure(func(event *remotemetrics.MeshTimeSyncChangedEvent) {
		sendSyncStatusChangedEvent(event)
	}))
}

func configureSchedulerQueryMetrics() {
	if Parameters.MetricsLevel > Info {
		return
	}
	remotemetrics.Events.SchedulerQuery.Attach(event.NewClosure(func(event *remotemetrics.SchedulerQueryEvent) { obtainSchedulerStats(event.Time) }))
}

func configureConflictConfirmationMetrics() {
	if Parameters.MetricsLevel > Info {
		return
	}
	deps.Mesh.Ledger.ConflictDAG.Events.ConflictAccepted.Attach(event.NewClosure(func(event *conflictdag.ConflictAcceptedEvent[utxo.TransactionID]) {
		onConflictConfirmed(event.ID)
	}))

	deps.Mesh.Ledger.ConflictDAG.Events.ConflictCreated.Attach(event.NewClosure(func(event *conflictdag.ConflictCreatedEvent[utxo.TransactionID, utxo.OutputID]) {
		activeConflictsMutex.Lock()
		defer activeConflictsMutex.Unlock()

		conflictID := event.ID
		if _, exists := activeConflicts[conflictID]; !exists {
			conflictTotalCountDB.Inc()
			activeConflicts[conflictID] = types.Void
			sendConflictMetrics()
		}
	}))
}

func configureBlockFinalizedMetrics() {
	if Parameters.MetricsLevel > Info {
		return
	} else if Parameters.MetricsLevel == Info {
		deps.Mesh.Ledger.Events.TransactionAccepted.Attach(event.NewClosure(func(event *ledger.TransactionAcceptedEvent) {
			onTransactionConfirmed(event.TransactionID)
		}))
	} else {
		deps.Mesh.ConfirmationOracle.Events().BlockAccepted.Attach(event.NewClosure(func(event *mesh_old.BlockAcceptedEvent) {
			onBlockFinalized(event.Block)
		}))
	}
}

func configureBlockScheduledMetrics() {
	if Parameters.MetricsLevel > Info {
		return
	} else if Parameters.MetricsLevel == Info {
		deps.Mesh.Scheduler.Events.BlockDiscarded.Attach(event.NewClosure(func(event *mesh_old.BlockDiscardedEvent) {
			sendBlockSchedulerRecord(event.BlockID, "blockDiscarded")
		}))
	} else {
		deps.Mesh.Scheduler.Events.BlockScheduled.Attach(event.NewClosure(func(event *mesh_old.BlockScheduledEvent) {
			sendBlockSchedulerRecord(event.BlockID, "blockScheduled")
		}))
		deps.Mesh.Scheduler.Events.BlockDiscarded.Attach(event.NewClosure(func(event *mesh_old.BlockDiscardedEvent) {
			sendBlockSchedulerRecord(event.BlockID, "blockDiscarded")
		}))
	}
}

func configureMissingBlockMetrics() {
	if Parameters.MetricsLevel > Info {
		return
	}

	deps.Mesh.Solidifier.Events.BlockMissing.Attach(event.NewClosure(func(event *mesh_old.BlockMissingEvent) {
		sendMissingBlockRecord(event.BlockID, "missingBlock")
	}))
	deps.Mesh.Storage.Events.MissingBlockStored.Attach(event.NewClosure(func(event *mesh_old.MissingBlockStoredEvent) {
		sendMissingBlockRecord(event.BlockID, "missingBlockStored")
	}))
}
