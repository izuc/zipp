package dashboard

import (
	"context"

	"github.com/izuc/zipp.foundation/app/daemon"
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/core/shutdown"
	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
)

func runLiveFeed(plugin *node.Plugin) {
	if err := daemon.BackgroundWorker("Dashboard[BlkUpdater]", func(ctx context.Context) {
		hook := deps.Protocol.Events.Engine.Mesh.BlockDAG.BlockAttached.Hook(func(block *blockdag.Block) {
			broadcastWsBlock(&wsblk{MsgTypeBlock, &blk{block.ID().Base58(), 0, block.Payload().Type()}})
		}, event.WithWorkerPool(plugin.WorkerPool))
		<-ctx.Done()
		log.Info("Stopping Dashboard[BlkUpdater] ...")
		hook.Unhook()
		log.Info("Stopping Dashboard[BlkUpdater] ... done")
	}, shutdown.PriorityDashboard); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}
