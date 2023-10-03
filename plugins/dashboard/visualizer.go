package dashboard

import (
	"context"
	"net/http"
	"sync"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/workerpool"
	"github.com/labstack/echo"

	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/mesh_old"

	"github.com/izuc/zipp/packages/node/shutdown"
)

var (
	visualizerWorkerCount     = 1
	visualizerWorkerQueueSize = 500
	visualizerWorkerPool      *workerpool.NonBlockingQueuedWorkerPool

	blkHistoryMutex    sync.RWMutex
	blkFinalized       map[string]bool
	blkHistory         []*mesh_old.Block
	maxBlkHistorySize  = 1000
	numHistoryToRemove = 100
)

// vertex defines a vertex in a DAG.
type vertex struct {
	ID              string              `json:"id"`
	ParentIDsByType map[string][]string `json:"parentIDsByType"`
	IsFinalized     bool                `json:"is_finalized"`
	IsTx            bool                `json:"is_tx"`
}

// tipinfo holds information about whether a given block is a tip or not.
type tipinfo struct {
	ID    string `json:"id"`
	IsTip bool   `json:"is_tip"`
}

// history holds a set of vertices in a DAG.
type history struct {
	Vertices []vertex `json:"vertices"`
}

func configureVisualizer() {
	visualizerWorkerPool = workerpool.NewNonBlockingQueuedWorkerPool(func(task workerpool.Task) {
		switch x := task.Param(0).(type) {
		case *mesh_old.Block:
			sendVertex(x, task.Param(1).(bool))
		case *mesh_old.TipEvent:
			sendTipInfo(task.Param(1).(mesh_old.BlockID), task.Param(2).(bool))
		}

		task.Return(nil)
	}, workerpool.WorkerCount(visualizerWorkerCount), workerpool.QueueSize(visualizerWorkerQueueSize))

	// configure blkHistory, blkSolid
	blkFinalized = make(map[string]bool, maxBlkHistorySize)
	blkHistory = make([]*mesh_old.Block, 0, maxBlkHistorySize)
}

func sendVertex(blk *mesh_old.Block, finalized bool) {
	broadcastWsBlock(&wsblk{MsgTypeVertex, &vertex{
		ID:              blk.ID().Base58(),
		ParentIDsByType: prepareParentReferences(blk),
		IsFinalized:     finalized,
		IsTx:            blk.Payload().Type() == devnetvm.TransactionType,
	}}, true)
}

func sendTipInfo(blockID mesh_old.BlockID, isTip bool) {
	broadcastWsBlock(&wsblk{MsgTypeTipInfo, &tipinfo{
		ID:    blockID.Base58(),
		IsTip: isTip,
	}}, true)
}

func runVisualizer() {
	processBlock := func(block *mesh_old.Block) {
		finalized := deps.Mesh.ConfirmationOracle.IsBlockConfirmed(block.ID())
		addToHistory(block, finalized)
		visualizerWorkerPool.TrySubmit(block, finalized)
	}

	notifyNewBlkStored := event.NewClosure(func(event *mesh_old.BlockStoredEvent) {
		processBlock(event.Block)
	})

	notifyNewBlkAccepted := event.NewClosure(func(event *mesh_old.BlockAcceptedEvent) {
		processBlock(event.Block)
	})

	notifyNewTip := event.NewClosure(func(tipEvent *mesh_old.TipEvent) {
		visualizerWorkerPool.TrySubmit(tipEvent, tipEvent.BlockID, true)
	})

	notifyDeletedTip := event.NewClosure(func(tipEvent *mesh_old.TipEvent) {
		visualizerWorkerPool.TrySubmit(tipEvent, tipEvent.BlockID, false)
	})

	if err := daemon.BackgroundWorker("Dashboard[Visualizer]", func(ctx context.Context) {
		deps.Mesh.Storage.Events.BlockStored.Attach(notifyNewBlkStored)
		defer deps.Mesh.Storage.Events.BlockStored.Detach(notifyNewBlkStored)
		deps.Mesh.ConfirmationOracle.Events().BlockAccepted.Attach(notifyNewBlkAccepted)
		defer deps.Mesh.ConfirmationOracle.Events().BlockAccepted.Detach(notifyNewBlkAccepted)
		deps.Mesh.TipManager.Events.TipAdded.Attach(notifyNewTip)
		defer deps.Mesh.TipManager.Events.TipAdded.Detach(notifyNewTip)
		deps.Mesh.TipManager.Events.TipRemoved.Attach(notifyDeletedTip)
		defer deps.Mesh.TipManager.Events.TipRemoved.Detach(notifyDeletedTip)
		<-ctx.Done()
		log.Info("Stopping Dashboard[Visualizer] ...")
		visualizerWorkerPool.Stop()
		log.Info("Stopping Dashboard[Visualizer] ... done")
	}, shutdown.PriorityDashboard); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}

func setupVisualizerRoutes(routeGroup *echo.Group) {
	routeGroup.GET("/visualizer/history", func(c echo.Context) (err error) {
		blkHistoryMutex.RLock()
		defer blkHistoryMutex.RUnlock()

		cpyHistory := make([]*mesh_old.Block, len(blkHistory))
		copy(cpyHistory, blkHistory)

		var res []vertex
		for _, blk := range cpyHistory {
			res = append(res, vertex{
				ID:              blk.ID().Base58(),
				ParentIDsByType: prepareParentReferences(blk),
				IsFinalized:     blkFinalized[blk.ID().Base58()],
				IsTx:            blk.Payload().Type() == devnetvm.TransactionType,
			})
		}

		return c.JSON(http.StatusOK, history{Vertices: res})
	})
}

func addToHistory(blk *mesh_old.Block, finalized bool) {
	blkHistoryMutex.Lock()
	defer blkHistoryMutex.Unlock()
	if _, exist := blkFinalized[blk.ID().Base58()]; exist {
		blkFinalized[blk.ID().Base58()] = finalized
		return
	}

	// remove 100 old blks if the slice is full
	if len(blkHistory) >= maxBlkHistorySize {
		for i := 0; i < numHistoryToRemove; i++ {
			delete(blkFinalized, blkHistory[i].ID().Base58())
		}
		blkHistory = append(blkHistory[:0], blkHistory[numHistoryToRemove:maxBlkHistorySize]...)
	}
	// add new blk
	blkHistory = append(blkHistory, blk)
	blkFinalized[blk.ID().Base58()] = finalized
}
