package dagsvisualizer

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/generics/lo"
	"github.com/izuc/zipp.foundation/core/generics/set"
	"github.com/izuc/zipp.foundation/core/generics/walker"
	"github.com/izuc/zipp.foundation/core/types/confirmation"
	"github.com/izuc/zipp.foundation/core/workerpool"
	"github.com/labstack/echo"

	"github.com/izuc/zipp/packages/app/jsonmodels"
	"github.com/izuc/zipp/packages/node/shutdown"

	"github.com/izuc/zipp/packages/core/conflictdag"
	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

var (
	visualizerWorkerCount     = 1
	visualizerWorkerQueueSize = 500
	visualizerWorkerPool      *workerpool.NonBlockingQueuedWorkerPool
	maxWsBlockBufferSize      = 200
	buffer                    []*wsBlock
	bufferMutex               sync.RWMutex
)

func setupVisualizer() {
	// create and start workerpool
	visualizerWorkerPool = workerpool.NewNonBlockingQueuedWorkerPool(func(task workerpool.Task) {
		broadcastWsBlock(task.Param(0))
	}, workerpool.WorkerCount(visualizerWorkerCount), workerpool.QueueSize(visualizerWorkerQueueSize))
}

func runVisualizer() {
	if err := daemon.BackgroundWorker("Dags Visualizer[Visualizer]", func(ctx context.Context) {
		// register to events
		registerMeshEvents()
		registerUTXOEvents()
		registerConflictEvents()

		<-ctx.Done()
		log.Info("Stopping DAGs Visualizer ...")
		visualizerWorkerPool.Stop()
		log.Info("Stopping DAGs Visualizer ... done")
	}, shutdown.PriorityDashboard); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}

func registerMeshEvents() {
	storeClosure := event.NewClosure(func(event *mesh_old.BlockStoredEvent) {
		wsBlk := &wsBlock{
			Type: BlkTypeMeshVertex,
			Data: newMeshVertex(event.Block),
		}
		visualizerWorkerPool.TrySubmit(wsBlk)
		storeWsBlock(wsBlk)
	})

	bookedClosure := event.NewClosure(func(event *mesh_old.BlockBookedEvent) {
		blockID := event.BlockID
		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blkMetadata *mesh_old.BlockMetadata) {
			conflictIDs, err := deps.Mesh.Booker.BlockConflictIDs(blockID)
			if err != nil {
				conflictIDs = set.NewAdvancedSet[utxo.TransactionID]()
			}

			wsBlk := &wsBlock{
				Type: BlkTypeMeshBooked,
				Data: &meshBooked{
					ID:          blockID.Base58(),
					IsMarker:    blkMetadata.StructureDetails().IsPastMarker(),
					ConflictIDs: lo.Map(conflictIDs.Slice(), utxo.TransactionID.Base58),
				},
			}
			visualizerWorkerPool.TrySubmit(wsBlk)
			storeWsBlock(wsBlk)
		})
	})

	blkConfirmedClosure := event.NewClosure(func(event *mesh_old.BlockAcceptedEvent) {
		blockID := event.Block.ID()
		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blkMetadata *mesh_old.BlockMetadata) {
			wsBlk := &wsBlock{
				Type: BlkTypeMeshConfirmed,
				Data: &meshConfirmed{
					ID:                    blockID.Base58(),
					ConfirmationState:     blkMetadata.ConfirmationState().String(),
					ConfirmationStateTime: blkMetadata.ConfirmationStateTime().UnixNano(),
				},
			}
			visualizerWorkerPool.TrySubmit(wsBlk)
			storeWsBlock(wsBlk)
		})
	})

	txAcceptedClosure := event.NewClosure(func(event *ledger.TransactionAcceptedEvent) {
		var blkID mesh_old.BlockID
		deps.Mesh.Storage.Attachments(event.TransactionID).Consume(func(a *mesh_old.Attachment) {
			blkID = a.BlockID()
		})

		wsBlk := &wsBlock{
			Type: BlkTypeMeshTxConfirmationState,
			Data: &meshTxConfirmationStateChanged{
				ID:          blkID.Base58(),
				IsConfirmed: deps.AcceptanceGadget.IsTransactionConfirmed(event.TransactionID),
			},
		}
		visualizerWorkerPool.TrySubmit(wsBlk)
		storeWsBlock(wsBlk)
	})

	deps.Mesh.Storage.Events.BlockStored.Attach(storeClosure)
	deps.Mesh.Booker.Events.BlockBooked.Attach(bookedClosure)
	deps.AcceptanceGadget.Events().BlockAccepted.Attach(blkConfirmedClosure)
	deps.Mesh.Ledger.Events.TransactionAccepted.Attach(txAcceptedClosure)
}

func registerUTXOEvents() {
	storeClosure := event.NewClosure(func(event *mesh_old.BlockStoredEvent) {
		if event.Block.Payload().Type() == devnetvm.TransactionType {
			tx := event.Block.Payload().(*devnetvm.Transaction)
			wsBlk := &wsBlock{
				Type: BlkTypeUTXOVertex,
				Data: newUTXOVertex(event.Block.ID(), tx),
			}
			visualizerWorkerPool.TrySubmit(wsBlk)
			storeWsBlock(wsBlk)
		}
	})

	bookedClosure := event.NewClosure(func(event *mesh_old.BlockBookedEvent) {
		blockID := event.BlockID
		deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
			if block.Payload().Type() == devnetvm.TransactionType {
				tx := block.Payload().(*devnetvm.Transaction)
				deps.Mesh.Ledger.Storage.CachedTransactionMetadata(tx.ID()).Consume(func(txMetadata *ledger.TransactionMetadata) {
					wsBlk := &wsBlock{
						Type: BlkTypeUTXOBooked,
						Data: &utxoBooked{
							ID:          tx.ID().Base58(),
							ConflictIDs: lo.Map(txMetadata.ConflictIDs().Slice(), utxo.TransactionID.Base58),
						},
					}
					visualizerWorkerPool.TrySubmit(wsBlk)
					storeWsBlock(wsBlk)
				})
			}
		})
	})

	txAcceptedClosure := event.NewClosure(func(event *ledger.TransactionAcceptedEvent) {
		txID := event.TransactionID
		deps.Mesh.Ledger.Storage.CachedTransactionMetadata(txID).Consume(func(txMetadata *ledger.TransactionMetadata) {
			wsBlk := &wsBlock{
				Type: BlkTypeUTXOConfirmationStateChanged,
				Data: &utxoConfirmationStateChanged{
					ID:                    txID.Base58(),
					ConfirmationState:     txMetadata.ConfirmationState().String(),
					ConfirmationStateTime: txMetadata.ConfirmationStateTime().UnixNano(),
					IsConfirmed:           deps.AcceptanceGadget.IsTransactionConfirmed(txID),
				},
			}
			visualizerWorkerPool.TrySubmit(wsBlk)
			storeWsBlock(wsBlk)
		})
	})

	deps.Mesh.Storage.Events.BlockStored.Attach(storeClosure)
	deps.Mesh.Booker.Events.BlockBooked.Attach(bookedClosure)
	deps.Mesh.Ledger.Events.TransactionAccepted.Attach(txAcceptedClosure)
}

func registerConflictEvents() {
	createdClosure := event.NewClosure(func(event *conflictdag.ConflictCreatedEvent[utxo.TransactionID, utxo.OutputID]) {
		wsBlk := &wsBlock{
			Type: BlkTypeConflictVertex,
			Data: newConflictVertex(event.ID),
		}
		visualizerWorkerPool.TrySubmit(wsBlk)
		storeWsBlock(wsBlk)
	})

	parentUpdateClosure := event.NewClosure(func(event *conflictdag.ConflictParentsUpdatedEvent[utxo.TransactionID, utxo.OutputID]) {
		lo.Map(event.ParentsConflictIDs.Slice(), utxo.TransactionID.Base58)
		wsBlk := &wsBlock{
			Type: BlkTypeConflictParentsUpdate,
			Data: &conflictParentUpdate{
				ID:      event.ConflictID.Base58(),
				Parents: lo.Map(event.ParentsConflictIDs.Slice(), utxo.TransactionID.Base58),
			},
		}
		visualizerWorkerPool.TrySubmit(wsBlk)
		storeWsBlock(wsBlk)
	})

	conflictConfirmedClosure := event.NewClosure(func(event *conflictdag.ConflictAcceptedEvent[utxo.TransactionID]) {
		wsBlk := &wsBlock{
			Type: BlkTypeConflictConfirmationStateChanged,
			Data: &conflictConfirmationStateChanged{
				ID:                event.ID.Base58(),
				ConfirmationState: confirmation.Accepted.String(),
				IsConfirmed:       true,
			},
		}
		visualizerWorkerPool.TrySubmit(wsBlk)
		storeWsBlock(wsBlk)
	})

	conflictWeightChangedClosure := event.NewClosure(func(e *mesh_old.ConflictWeightChangedEvent) {
		conflictConfirmationState := deps.Mesh.Ledger.ConflictDAG.ConfirmationState(utxo.NewTransactionIDs(e.ConflictID))
		wsBlk := &wsBlock{
			Type: BlkTypeConflictWeightChanged,
			Data: &conflictWeightChanged{
				ID:                e.ConflictID.Base58(),
				Weight:            e.Weight,
				ConfirmationState: conflictConfirmationState.String(),
			},
		}
		visualizerWorkerPool.TrySubmit(wsBlk)
		storeWsBlock(wsBlk)
	})

	deps.Mesh.Ledger.ConflictDAG.Events.ConflictCreated.Attach(createdClosure)
	deps.Mesh.Ledger.ConflictDAG.Events.ConflictAccepted.Attach(conflictConfirmedClosure)
	deps.Mesh.Ledger.ConflictDAG.Events.ConflictParentsUpdated.Attach(parentUpdateClosure)
	deps.Mesh.ApprovalWeightManager.Events.ConflictWeightChanged.Attach(conflictWeightChangedClosure)
}

func setupDagsVisualizerRoutes(routeGroup *echo.Group) {
	routeGroup.GET("/dagsvisualizer/conflict/:conflictID", func(c echo.Context) (err error) {
		parents := make(map[string]*conflictVertex)
		var conflictID utxo.TransactionID
		if err = conflictID.FromBase58(c.Param("conflictID")); err != nil {
			err = c.JSON(http.StatusBadRequest, jsonmodels.NewErrorResponse(err))
			return
		}
		vertex := newConflictVertex(conflictID)
		parents[vertex.ID] = vertex
		getConflictsToMaster(vertex, parents)

		var conflicts []*conflictVertex
		for _, conflict := range parents {
			conflicts = append(conflicts, conflict)
		}
		return c.JSON(http.StatusOK, conflicts)
	})

	routeGroup.GET("/dagsvisualizer/search/:start/:end", func(c echo.Context) (err error) {
		startTimestamp := parseStringToTimestamp(c.Param("start"))
		endTimestamp := parseStringToTimestamp(c.Param("end"))

		reqValid := isTimeIntervalValid(startTimestamp, endTimestamp)
		if !reqValid {
			return c.JSON(http.StatusBadRequest, searchResult{Error: "invalid timestamp range"})
		}

		blocks := []*meshVertex{}
		txs := []*utxoVertex{}
		conflicts := []*conflictVertex{}
		conflictMap := set.NewAdvancedSet[utxo.TransactionID]()
		entryBlks := mesh_old.NewBlockIDs()
		deps.Mesh.Storage.Children(mesh_old.EmptyBlockID).Consume(func(child *mesh_old.Child) {
			entryBlks.Add(child.ChildBlockID())
		})

		deps.Mesh.Utils.WalkBlockID(func(blockID mesh_old.BlockID, walker *walker.Walker[mesh_old.BlockID]) {
			deps.Mesh.Storage.Block(blockID).Consume(func(blk *mesh_old.Block) {
				// only keep blocks that is issued in the given time interval
				if blk.IssuingTime().After(startTimestamp) && blk.IssuingTime().Before(endTimestamp) {
					// add block
					meshNode := newMeshVertex(blk)
					blocks = append(blocks, meshNode)

					// add tx
					if meshNode.IsTx {
						utxoNode := newUTXOVertex(blk.ID(), blk.Payload().(*devnetvm.Transaction))
						txs = append(txs, utxoNode)
					}

					// add conflict
					conflictIDs, err := deps.Mesh.Booker.BlockConflictIDs(blk.ID())
					if err != nil {
						conflictIDs = set.NewAdvancedSet[utxo.TransactionID]()
					}
					for it := conflictIDs.Iterator(); it.HasNext(); {
						conflictID := it.Next()
						if conflictMap.Has(conflictID) {
							continue
						}

						conflictMap.Add(conflictID)
						conflicts = append(conflicts, newConflictVertex(conflictID))
					}
				}

				// continue walking if the block is issued before endTimestamp
				if blk.IssuingTime().Before(endTimestamp) {
					deps.Mesh.Storage.Children(blockID).Consume(func(child *mesh_old.Child) {
						walker.Push(child.ChildBlockID())
					})
				}
			})
		}, entryBlks)

		return c.JSON(http.StatusOK, searchResult{Blocks: blocks, Txs: txs, Conflicts: conflicts})
	})
}

func parseStringToTimestamp(str string) (t time.Time) {
	ts, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

func isTimeIntervalValid(start, end time.Time) (valid bool) {
	if start.IsZero() || end.IsZero() {
		return false
	}

	if start.After(end) {
		return false
	}

	return true
}

func newMeshVertex(block *mesh_old.Block) (ret *meshVertex) {
	deps.Mesh.Storage.BlockMetadata(block.ID()).Consume(func(blkMetadata *mesh_old.BlockMetadata) {
		conflictIDs, err := deps.Mesh.Booker.BlockConflictIDs(block.ID())
		if err != nil {
			conflictIDs = set.NewAdvancedSet[utxo.TransactionID]()
		}
		ret = &meshVertex{
			ID:                    block.ID().Base58(),
			StrongParentIDs:       block.ParentsByType(mesh_old.StrongParentType).Base58(),
			WeakParentIDs:         block.ParentsByType(mesh_old.WeakParentType).Base58(),
			ShallowLikeParentIDs:  block.ParentsByType(mesh_old.ShallowLikeParentType).Base58(),
			ConflictIDs:           lo.Map(conflictIDs.Slice(), utxo.TransactionID.Base58),
			IsMarker:              blkMetadata.StructureDetails() != nil && blkMetadata.StructureDetails().IsPastMarker(),
			IsTx:                  block.Payload().Type() == devnetvm.TransactionType,
			IsConfirmed:           deps.AcceptanceGadget.IsBlockConfirmed(block.ID()),
			ConfirmationStateTime: blkMetadata.ConfirmationStateTime().UnixNano(),
			ConfirmationState:     blkMetadata.ConfirmationState().String(),
		}
	})

	if ret.IsTx {
		ret.TxID = block.Payload().(*devnetvm.Transaction).ID().Base58()
	}
	return
}

func newUTXOVertex(blkID mesh_old.BlockID, tx *devnetvm.Transaction) (ret *utxoVertex) {
	inputs := make([]*jsonmodels.Input, len(tx.Essence().Inputs()))
	for i, input := range tx.Essence().Inputs() {
		inputs[i] = jsonmodels.NewInput(input)
	}

	outputs := make([]string, len(tx.Essence().Outputs()))
	for i, output := range tx.Essence().Outputs() {
		outputs[i] = output.ID().Base58()
	}

	var confirmationState string
	var confirmedTime int64
	var conflictIDs []string
	deps.Mesh.Ledger.Storage.CachedTransactionMetadata(tx.ID()).Consume(func(txMetadata *ledger.TransactionMetadata) {
		confirmationState = txMetadata.ConfirmationState().String()
		confirmedTime = txMetadata.ConfirmationStateTime().UnixNano()
		conflictIDs = lo.Map(txMetadata.ConflictIDs().Slice(), utxo.TransactionID.Base58)
	})

	ret = &utxoVertex{
		BlkID:                 blkID.Base58(),
		ID:                    tx.ID().Base58(),
		Inputs:                inputs,
		Outputs:               outputs,
		IsConfirmed:           deps.AcceptanceGadget.IsTransactionConfirmed(tx.ID()),
		ConflictIDs:           conflictIDs,
		ConfirmationState:     confirmationState,
		ConfirmationStateTime: confirmedTime,
	}

	return ret
}

func newConflictVertex(conflictID utxo.TransactionID) (ret *conflictVertex) {
	deps.Mesh.Ledger.ConflictDAG.Storage.CachedConflict(conflictID).Consume(func(conflict *conflictdag.Conflict[utxo.TransactionID, utxo.OutputID]) {
		conflicts := make(map[utxo.OutputID][]utxo.TransactionID)
		// get conflicts of a conflict
		for it := conflict.ConflictSetIDs().Iterator(); it.HasNext(); {
			conflictID := it.Next()
			conflicts[conflictID] = make([]utxo.TransactionID, 0)
			deps.Mesh.Ledger.ConflictDAG.Storage.CachedConflictMembers(conflictID).Consume(func(conflictMember *conflictdag.ConflictMember[utxo.OutputID, utxo.TransactionID]) {
				conflicts[conflictID] = append(conflicts[conflictID], conflictMember.ConflictID())
			})
		}

		ret = &conflictVertex{
			ID:                conflictID.Base58(),
			Parents:           lo.Map(conflict.Parents().Slice(), utxo.TransactionID.Base58),
			Conflicts:         jsonmodels.NewGetConflictConflictsResponse(conflict.ID(), conflicts),
			IsConfirmed:       deps.AcceptanceGadget.IsConflictConfirmed(conflictID),
			ConfirmationState: deps.Mesh.Ledger.ConflictDAG.ConfirmationState(utxo.NewTransactionIDs(conflictID)).String(),
			AW:                deps.Mesh.ApprovalWeightManager.WeightOfConflict(conflictID),
		}
	})
	return
}

func storeWsBlock(blk *wsBlock) {
	bufferMutex.Lock()
	defer bufferMutex.Unlock()
	if len(buffer) >= maxWsBlockBufferSize {
		buffer = buffer[1:]
	}
	buffer = append(buffer, blk)
}

func getConflictsToMaster(vertex *conflictVertex, parents map[string]*conflictVertex) {
	for _, IDBase58 := range vertex.Parents {
		if _, ok := parents[IDBase58]; !ok {
			var ID utxo.TransactionID
			if err := ID.FromBase58(IDBase58); err == nil {
				parentVertex := newConflictVertex(ID)
				parents[parentVertex.ID] = parentVertex
				getConflictsToMaster(parentVertex, parents)
			}
		}
	}
}
