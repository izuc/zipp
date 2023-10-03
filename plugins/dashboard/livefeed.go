package dashboard

import (
	"context"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/workerpool"

	"github.com/izuc/zipp/packages/core/mesh_old"

	"github.com/izuc/zipp/packages/node/shutdown"
)

var (
	liveFeedWorkerCount     = 1
	liveFeedWorkerQueueSize = 50
	liveFeedWorkerPool      *workerpool.NonBlockingQueuedWorkerPool
)

func configureLiveFeed() {
	liveFeedWorkerPool = workerpool.NewNonBlockingQueuedWorkerPool(func(task workerpool.Task) {
		block := task.Param(0).(*mesh_old.Block)

		broadcastWsBlock(&wsblk{MsgTypeBlock, &blk{block.ID().Base58(), 0, uint32(block.Payload().Type())}})

		task.Return(nil)
	}, workerpool.WorkerCount(liveFeedWorkerCount), workerpool.QueueSize(liveFeedWorkerQueueSize))
}

func runLiveFeed() {
	notifyNewBlk := event.NewClosure(func(event *mesh_old.BlockStoredEvent) {
		liveFeedWorkerPool.TrySubmit(event.Block)
	})

	if err := daemon.BackgroundWorker("Dashboard[BlkUpdater]", func(ctx context.Context) {
		deps.Mesh.Storage.Events.BlockStored.Attach(notifyNewBlk)
		<-ctx.Done()
		log.Info("Stopping Dashboard[BlkUpdater] ...")
		deps.Mesh.Storage.Events.BlockStored.Detach(notifyNewBlk)
		liveFeedWorkerPool.Stop()
		log.Info("Stopping Dashboard[BlkUpdater] ... done")
	}, shutdown.PriorityDashboard); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}
