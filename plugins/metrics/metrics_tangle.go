package metrics

import (
	"time"

	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/app/collector"
	"github.com/izuc/zipp/packages/network"
	"github.com/izuc/zipp/packages/protocol/congestioncontrol/icca/scheduler"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/blockgadget"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
	"github.com/izuc/zipp/packages/protocol/models"
)

const (
	meshNamespace = "mesh"

	tipsCount                     = "tips_count"
	blockPerTypeCount             = "block_per_type_total"
	missingBlocksCount            = "missing_block_total"
	parentPerTypeCount            = "parent_per_type_total"
	blocksPerComponentCount       = "blocks_per_component_total"
	timeSinceReceivedPerComponent = "time_since_received_per_component_seconds"
	requestQueueSize              = "request_queue_size"
	blocksOrphanedCount           = "blocks_orphaned_total"
	acceptedBlocksCount           = "accepted_blocks_count"
)

var MeshMetrics = collector.NewCollection(meshNamespace,
	collector.WithMetric(collector.NewMetric(tipsCount,
		collector.WithType(collector.Gauge),
		collector.WithHelp("Number of tips in the mesh"),
		collector.WithCollectFunc(func() map[string]float64 {
			count := deps.Protocol.TipManager.TipCount()
			return collector.SingleValue(count)
		}),
	)),
	collector.WithMetric(collector.NewMetric(blockPerTypeCount,
		collector.WithType(collector.GaugeVec),
		collector.WithHelp("Number of blocks per type in the mesh"),
		collector.WithLabels("type"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Engine.Consensus.BlockGadget.BlockAccepted.Hook(func(block *blockgadget.Block) {
				blockType := collector.NewBlockType(block.Payload().Type()).String()
				deps.Collector.Increment(meshNamespace, blockPerTypeCount, blockType)
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(missingBlocksCount,
		collector.WithType(collector.Counter),
		collector.WithHelp("Number of blocks missing during the solidification in the mesh"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Engine.Mesh.BlockDAG.BlockMissing.Hook(func(_ *blockdag.Block) {
				deps.Collector.Increment(meshNamespace, missingBlocksCount)
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(parentPerTypeCount,
		collector.WithType(collector.CounterVec),
		collector.WithHelp("Number of parents of the block per its type"),
		collector.WithLabels("type"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Engine.Consensus.BlockGadget.BlockAccepted.Hook(func(block *blockgadget.Block) {
				blockType := collector.NewBlockType(block.Payload().Type()).String()
				block.ForEachParent(func(parent models.Parent) {
					deps.Collector.Increment(meshNamespace, parentPerTypeCount, blockType)
				})
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(blocksPerComponentCount,
		collector.WithType(collector.CounterVec),
		collector.WithHelp("Number of blocks per component"),
		collector.WithLabels("component"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Network.BlockReceived.Hook(func(_ *network.BlockReceivedEvent) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Received.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.Engine.Filter.BlockAllowed.Hook(func(_ *models.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Allowed.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.BlockIssuer.Events.BlockIssued.Hook(func(_ *models.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Issued.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.Engine.Mesh.BlockDAG.BlockAttached.Hook(func(block *blockdag.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Attached.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.Engine.Mesh.BlockDAG.BlockSolid.Hook(func(block *blockdag.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Solidified.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.CongestionControl.Scheduler.BlockScheduled.Hook(func(block *scheduler.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Scheduled.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.Engine.Mesh.Booker.BlockBooked.Hook(func(_ *booker.BlockBookedEvent) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.Booked.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.CongestionControl.Scheduler.BlockDropped.Hook(func(block *scheduler.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.SchedulerDropped.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
			deps.Protocol.Events.CongestionControl.Scheduler.BlockSkipped.Hook(func(block *scheduler.Block) {
				deps.Collector.Increment(meshNamespace, blocksPerComponentCount, collector.SchedulerSkipped.String())
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(blocksOrphanedCount,
		collector.WithType(collector.Counter),
		collector.WithHelp("Number of orphaned blocks"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Engine.Mesh.BlockDAG.BlockOrphaned.Hook(func(block *blockdag.Block) {
				deps.Collector.Increment(meshNamespace, blocksOrphanedCount)
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(acceptedBlocksCount,
		collector.WithType(collector.Counter),
		collector.WithHelp("Number of accepted blocks"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Engine.Consensus.BlockGadget.BlockAccepted.Hook(func(block *blockgadget.Block) {
				deps.Collector.Increment(meshNamespace, acceptedBlocksCount)
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(timeSinceReceivedPerComponent,
		collector.WithType(collector.CounterVec),
		collector.WithHelp("Time since the block was received per component"),
		collector.WithLabels("component"),
		collector.WithInitFunc(func() {
			deps.Protocol.Events.Engine.Consensus.BlockGadget.BlockAccepted.Hook(func(block *blockgadget.Block) {
				blockType := collector.NewBlockType(block.Payload().Type()).String()
				timeSince := float64(time.Since(block.IssuingTime()).Milliseconds())
				deps.Collector.Update(meshNamespace, timeSinceReceivedPerComponent, collector.MultiLabelsValues([]string{blockType}, timeSince))
			}, event.WithWorkerPool(Plugin.WorkerPool))
		}),
	)),
	collector.WithMetric(collector.NewMetric(requestQueueSize,
		collector.WithType(collector.Gauge),
		collector.WithHelp("Number of blocks in the request queue"),
		collector.WithCollectFunc(func() map[string]float64 {
			return collector.SingleValue(float64(deps.Protocol.Engine().BlockRequester.QueueSize()))
		}),
	)),
)

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
