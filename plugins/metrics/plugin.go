package metrics

import (
	"context"
	"time"

	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/autopeering/selection"
	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/logger"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/izuc/zipp.foundation/core/timeutil"
	"github.com/izuc/zipp.foundation/core/types"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/conflictdag"
	"github.com/izuc/zipp/packages/node/clock"

	"github.com/izuc/zipp/packages/core/ledger/utxo"
	"github.com/izuc/zipp/packages/core/mana"
	"github.com/izuc/zipp/packages/core/mesh_old"

	"github.com/izuc/zipp/packages/app/metrics"
	"github.com/izuc/zipp/packages/core/notarization"
	"github.com/izuc/zipp/packages/node/p2p"
	"github.com/izuc/zipp/packages/node/shutdown"
	"github.com/izuc/zipp/plugins/analysis/server"
)

// PluginName is the name of the metrics plugin.
const PluginName = "Metrics"

var (
	// Plugin is the plugin instance of the metrics plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
	log    *logger.Logger
)

type dependencies struct {
	dig.In

	Mesh          *mesh_old.Mesh
	P2Pmgr          *p2p.Manager        `optional:"true"`
	Selection       *selection.Protocol `optional:"true"`
	Local           *peer.Local
	NotarizationMgr *notarization.Manager
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure, run)
}

func configure(_ *node.Plugin) {
	log = logger.NewLogger(PluginName)
}

func run(_ *node.Plugin) {
	log.Infof("Starting %s ...", PluginName)
	if Parameters.Local {
		// initial measurement, since we have to know how many blocks are there in the db
		measureInitialDBStats()
		measureInitialConflictStats()
		registerLocalMetrics()
	}
	// Events from analysis server
	if Parameters.Global {
		server.Events.MetricHeartbeat.Attach(onMetricHeartbeatReceived)
	}

	// create a background worker that update the metrics every second
	if err := daemon.BackgroundWorker("Metrics Updater", func(ctx context.Context) {
		if Parameters.Local {
			// Do not block until the Ticker is shutdown because we might want to start multiple Tickers and we can
			// safely ignore the last execution when shutting down.
			timeutil.NewTicker(func() {
				measureCPUUsage()
				measureMemUsage()
				measureSynced()
				measureBlockTips()
				measureReceivedBPS()
				measureRequestQueueSize()
				measureGossipTraffic()
				measurePerComponentCounter()
				measureRateSetter()
				measureSchedulerMetrics()
			}, 1*time.Second, ctx)
		}

		if Parameters.Global {
			// Do not block until the Ticker is shutdown because we might want to start multiple Tickers and we can
			// safely ignore the last execution when shutting down.
			timeutil.NewTicker(calculateNetworkDiameter, 1*time.Minute, ctx)
		}

		// Wait before terminating so we get correct log blocks from the daemon regarding the shutdown order.
		<-ctx.Done()
	}, shutdown.PriorityMetrics); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}

	// create a background worker that updates the mana metrics
	if err := daemon.BackgroundWorker("Metrics Mana Updater", func(ctx context.Context) {
		if deps.P2Pmgr == nil {
			return
		}
		defer log.Infof("Stopping Metrics Mana Updater ... done")
		timeutil.NewTicker(func() {
			measureMana()
		}, Parameters.ManaUpdateInterval, ctx)
		// Wait before terminating so we get correct log blocks from the daemon regarding the shutdown order.
		<-ctx.Done()
		log.Infof("Stopping Metrics Mana Updater ...")
	}, shutdown.PriorityMetrics); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}

	if Parameters.ManaResearch {
		// create a background worker that updates the research mana metrics
		if err := daemon.BackgroundWorker("Metrics Research Mana Updater", func(ctx context.Context) {
			defer log.Infof("Stopping Metrics Research Mana Updater ... done")
			timeutil.NewTicker(func() {
				measureAccessResearchMana()
				measureConsensusResearchMana()
			}, Parameters.ManaUpdateInterval, ctx)
			// Wait before terminating so we get correct log blocks from the daemon regarding the shutdown order.
			<-ctx.Done()
			log.Infof("Stopping Metrics Research Mana Updater ...")
		}, shutdown.PriorityMetrics); err != nil {
			log.Panicf("Failed to start as daemon: %s", err)
		}
	}
}

func registerLocalMetrics() {
	// // Events declared in other packages which we want to listen to here ////

	// increase received BPS counter whenever we attached a block
	deps.Mesh.Storage.Events.BlockStored.Attach(event.NewClosure(func(event *mesh_old.BlockStoredEvent) {
		sumTimeMutex.Lock()
		defer sumTimeMutex.Unlock()
		increaseReceivedBPSCounter()
		increasePerPayloadCounter(event.Block.Payload().Type())

		deps.Mesh.Storage.BlockMetadata(event.Block.ID()).Consume(func(blkMetaData *mesh_old.BlockMetadata) {
			sumTimesSinceIssued[Store] += blkMetaData.ReceivedTime().Sub(event.Block.IssuingTime())
		})
		increasePerComponentCounter(Store)
	}))

	// blocks can only become solid once, then they stay like that, hence no .Dec() part
	deps.Mesh.Solidifier.Events.BlockSolid.Attach(event.NewClosure(func(event *mesh_old.BlockSolidEvent) {
		increasePerComponentCounter(Solidifier)
		sumTimeMutex.Lock()
		defer sumTimeMutex.Unlock()

		// Consume should release cachedBlockMetadata
		deps.Mesh.Storage.BlockMetadata(event.Block.ID()).Consume(func(blkMetaData *mesh_old.BlockMetadata) {
			if blkMetaData.IsSolid() {
				sumTimesSinceReceived[Solidifier] += blkMetaData.SolidificationTime().Sub(blkMetaData.ReceivedTime())
			}
		})
	}))

	// fired when a block gets added to missing block storage
	deps.Mesh.Solidifier.Events.BlockMissing.Attach(event.NewClosure(func(_ *mesh_old.BlockMissingEvent) {
		missingBlockCountDB.Inc()
		solidificationRequests.Inc()
	}))

	// fired when a missing block was received and removed from missing block storage
	deps.Mesh.Storage.Events.MissingBlockStored.Attach(event.NewClosure(func(_ *mesh_old.MissingBlockStoredEvent) {
		missingBlockCountDB.Dec()
	}))

	deps.Mesh.Scheduler.Events.BlockScheduled.Attach(event.NewClosure(func(event *mesh_old.BlockScheduledEvent) {
		increasePerComponentCounter(Scheduler)
		sumTimeMutex.Lock()
		defer sumTimeMutex.Unlock()
		schedulerTimeMutex.Lock()
		defer schedulerTimeMutex.Unlock()

		blockID := event.BlockID
		// Consume should release cachedBlockMetadata
		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blkMetaData *mesh_old.BlockMetadata) {
			if blkMetaData.Scheduled() {
				sumSchedulerBookedTime += blkMetaData.ScheduledTime().Sub(blkMetaData.BookedTime())

				sumTimesSinceReceived[Scheduler] += blkMetaData.ScheduledTime().Sub(blkMetaData.ReceivedTime())
				deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
					sumTimesSinceIssued[Scheduler] += blkMetaData.ScheduledTime().Sub(block.IssuingTime())
				})
			}
		})
	}))

	deps.Mesh.Booker.Events.BlockBooked.Attach(event.NewClosure(func(event *mesh_old.BlockBookedEvent) {
		increasePerComponentCounter(Booker)
		sumTimeMutex.Lock()
		defer sumTimeMutex.Unlock()

		blockID := event.BlockID
		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blkMetaData *mesh_old.BlockMetadata) {
			if blkMetaData.IsBooked() {
				sumTimesSinceReceived[Booker] += blkMetaData.BookedTime().Sub(blkMetaData.ReceivedTime())
				deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
					sumTimesSinceIssued[Booker] += blkMetaData.BookedTime().Sub(block.IssuingTime())
				})
			}
		})
	}))

	deps.Mesh.Scheduler.Events.BlockDiscarded.Attach(event.NewClosure(func(event *mesh_old.BlockDiscardedEvent) {
		increasePerComponentCounter(SchedulerDropped)
		sumTimeMutex.Lock()
		defer sumTimeMutex.Unlock()

		blockID := event.BlockID
		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blkMetaData *mesh_old.BlockMetadata) {
			sumTimesSinceReceived[SchedulerDropped] += clock.Since(blkMetaData.ReceivedTime())
			deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
				sumTimesSinceIssued[SchedulerDropped] += clock.Since(block.IssuingTime())
			})
		})
	}))

	deps.Mesh.Scheduler.Events.BlockSkipped.Attach(event.NewClosure(func(event *mesh_old.BlockSkippedEvent) {
		increasePerComponentCounter(SchedulerSkipped)
		sumTimeMutex.Lock()
		defer sumTimeMutex.Unlock()

		blockID := event.BlockID
		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blkMetaData *mesh_old.BlockMetadata) {
			sumTimesSinceReceived[SchedulerSkipped] += clock.Since(blkMetaData.ReceivedTime())
			deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
				sumTimesSinceIssued[SchedulerSkipped] += clock.Since(block.IssuingTime())
			})
		})
	}))

	deps.Mesh.ConfirmationOracle.Events().BlockAccepted.Attach(event.NewClosure(func(event *mesh_old.BlockAcceptedEvent) {
		blockType := DataBlock
		block := event.Block
		blockID := block.ID()
		deps.Mesh.Utils.ComputeIfTransaction(blockID, func(_ utxo.TransactionID) {
			blockType = Transaction
		})
		blockFinalizationTotalTimeMutex.Lock()
		defer blockFinalizationTotalTimeMutex.Unlock()
		finalizedBlockCountMutex.Lock()
		defer finalizedBlockCountMutex.Unlock()

		block.ForEachParent(func(parent mesh_old.Parent) {
			increasePerParentType(parent.Type)
		})
		blockFinalizationIssuedTotalTime[blockType] += uint64(clock.Since(block.IssuingTime()).Milliseconds())
		if deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
			blockFinalizationReceivedTotalTime[blockType] += uint64(clock.Since(blockMetadata.ReceivedTime()).Milliseconds())
		}) {
			finalizedBlockCount[blockType]++
		}
	}))

	deps.Mesh.Ledger.ConflictDAG.Events.ConflictAccepted.Attach(event.NewClosure(func(event *conflictdag.ConflictAcceptedEvent[utxo.TransactionID]) {
		activeConflictsMutex.Lock()
		defer activeConflictsMutex.Unlock()

		conflictID := event.ID
		if _, exists := activeConflicts[conflictID]; !exists {
			return
		}
		oldestAttachmentTime, _, err := deps.Mesh.Utils.FirstAttachment(conflictID)
		if err != nil {
			return
		}
		deps.Mesh.Ledger.ConflictDAG.Utils.ForEachConflictingConflictID(conflictID, func(conflictingConflictID utxo.TransactionID) bool {
			if _, exists := activeConflicts[conflictID]; exists && conflictingConflictID != conflictID {
				finalizedConflictCountDB.Inc()
				delete(activeConflicts, conflictingConflictID)
			}
			return true
		})
		finalizedConflictCountDB.Inc()
		confirmedConflictCount.Inc()
		conflictConfirmationTotalTime.Add(uint64(clock.Since(oldestAttachmentTime).Milliseconds()))

		delete(activeConflicts, conflictID)
	}))

	deps.Mesh.Ledger.ConflictDAG.Events.ConflictCreated.Attach(event.NewClosure(func(event *conflictdag.ConflictCreatedEvent[utxo.TransactionID, utxo.OutputID]) {
		activeConflictsMutex.Lock()
		defer activeConflictsMutex.Unlock()

		conflictID := event.ID
		if _, exists := activeConflicts[conflictID]; !exists {
			conflictTotalCountDB.Inc()
			activeConflicts[conflictID] = types.Void
		}
	}))

	metrics.Events.AnalysisOutboundBytes.Attach(event.NewClosure(func(event *metrics.AnalysisOutboundBytesEvent) {
		analysisOutboundBytes.Add(event.AmountBytes)
	}))
	metrics.Events.CPUUsage.Attach(event.NewClosure(func(evnet *metrics.CPUUsageEvent) {
		cpuUsage.Store(evnet.CPUPercent)
	}))
	metrics.Events.MemUsage.Attach(event.NewClosure(func(event *metrics.MemUsageEvent) {
		memUsageBytes.Store(event.MemAllocBytes)
	}))

	deps.P2Pmgr.NeighborGroupEvents(p2p.NeighborsGroupAuto).NeighborRemoved.Attach(onNeighborRemoved)
	deps.P2Pmgr.NeighborGroupEvents(p2p.NeighborsGroupAuto).NeighborAdded.Attach(onNeighborAdded)

	if deps.Selection != nil {
		deps.Selection.Events().IncomingPeering.Hook(onAutopeeringSelection)
		deps.Selection.Events().OutgoingPeering.Hook(onAutopeeringSelection)
	}

	// mana pledge events
	mana.Events.Pledged.Attach(event.NewClosure(func(ev *mana.PledgedEvent) {
		addPledge(ev)
	}))

	deps.NotarizationMgr.Events.EpochCommittable.Attach(onEpochCommitted)
}
