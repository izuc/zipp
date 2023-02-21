package booker

import (
	"fmt"
	"time"

	"github.com/iotaledger/goshimmer/packages/core/epoch"
	"github.com/iotaledger/goshimmer/packages/protocol/ledger/utxo"
	"github.com/iotaledger/goshimmer/packages/protocol/models"
	"github.com/iotaledger/hive.go/ds/advancedset"
	"github.com/iotaledger/hive.go/ds/set"
	"github.com/iotaledger/hive.go/ds/shrinkingmap"
	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/runtime/syncutils"
)

type attachments struct {
	attachments *shrinkingmap.ShrinkingMap[utxo.TransactionID, *shrinkingmap.ShrinkingMap[epoch.Index, *shrinkingmap.ShrinkingMap[models.BlockID, *Block]]]
	evictionMap *shrinkingmap.ShrinkingMap[epoch.Index, set.Set[utxo.TransactionID]]

	// nonOrphanedCounter is used to count all non-orphaned attachment of a transaction,
	// so that it's not necessary to iterate through all the attachments to check if the transaction is orphaned.
	nonOrphanedCounter *shrinkingmap.ShrinkingMap[utxo.TransactionID, uint32]

	mutex *syncutils.DAGMutex[utxo.TransactionID]
}

func newAttachments() (newAttachments *attachments) {
	return &attachments{
		attachments:        shrinkingmap.New[utxo.TransactionID, *shrinkingmap.ShrinkingMap[epoch.Index, *shrinkingmap.ShrinkingMap[models.BlockID, *Block]]](),
		evictionMap:        shrinkingmap.New[epoch.Index, set.Set[utxo.TransactionID]](),
		nonOrphanedCounter: shrinkingmap.New[utxo.TransactionID, uint32](),

		mutex: syncutils.NewDAGMutex[utxo.TransactionID](),
	}
}

func (a *attachments) Store(txID utxo.TransactionID, block *Block) (created bool) {
	a.mutex.Lock(txID)
	defer a.mutex.Unlock(txID)

	if !a.storeAttachment(txID, block) {
		return false
	}

	prevValue, _ := a.nonOrphanedCounter.GetOrCreate(txID, func() uint32 { return 0 })
	a.nonOrphanedCounter.Set(txID, prevValue+1)

	a.updateEvictionMap(block.ID().EpochIndex, txID)

	return true
}

func (a *attachments) AttachmentOrphaned(txID utxo.TransactionID, block *Block) (attachmentBlock *Block, attachmentOrphaned, lastAttachmentOrphaned bool) {
	a.mutex.Lock(txID)
	defer a.mutex.Unlock(txID)

	storage := a.storage(txID, false)
	if storage == nil {
		// zero attachments registered
		return nil, false, false
	}

	attachmentsOfEpoch, exists := storage.Get(block.ID().Index())
	if !exists {
		// there is no attachments of the transaction in the epoch, so no need to continue
		return nil, false, false
	}

	attachmentBlock, attachmentExists := attachmentsOfEpoch.Get(block.ID())
	if !attachmentExists {
		return nil, false, false
	}

	prevValue, counterExists := a.nonOrphanedCounter.Get(txID)
	if !counterExists {
		panic(fmt.Sprintf("non orphaned attachment counter does not exist for TxID(%s)", txID.String()))
	}

	newValue := prevValue - 1
	a.nonOrphanedCounter.Set(txID, newValue)

	return attachmentBlock, true, newValue == 0
}

func (a *attachments) Get(txID utxo.TransactionID) (attachments []*Block) {
	if txStorage := a.storage(txID, false); txStorage != nil {
		txStorage.ForEach(func(_ epoch.Index, blocks *shrinkingmap.ShrinkingMap[models.BlockID, *Block]) bool {
			blocks.ForEach(func(_ models.BlockID, block *Block) bool {
				attachments = append(attachments, block)
				return true
			})
			return true
		})
	}

	return
}

func (a *attachments) GetAttachmentBlocks(txID utxo.TransactionID) (attachments *advancedset.AdvancedSet[*Block]) {
	attachments = advancedset.NewAdvancedSet[*Block]()

	if txStorage := a.storage(txID, false); txStorage != nil {
		txStorage.ForEach(func(_ epoch.Index, blocks *shrinkingmap.ShrinkingMap[models.BlockID, *Block]) bool {
			blocks.ForEach(func(_ models.BlockID, attachmentBlock *Block) bool {
				attachments.Add(attachmentBlock)
				return true
			})
			return true
		})
	}

	return
}

func (a *attachments) Evict(epochIndex epoch.Index) {
	if txIDs, exists := a.evictionMap.Get(epochIndex); exists {
		a.evictionMap.Delete(epochIndex)

		txIDs.ForEach(func(txID utxo.TransactionID) {
			a.mutex.Lock(txID)
			defer a.mutex.Unlock(txID)

			if attachmentsOfTX := a.storage(txID, false); attachmentsOfTX != nil && attachmentsOfTX.Delete(epochIndex) && attachmentsOfTX.Size() == 0 {
				a.attachments.Delete(txID)
			}
		})
	}
}

func (a *attachments) storeAttachment(txID utxo.TransactionID, block *Block) (created bool) {
	attachmentsOfEpoch, _ := a.storage(txID, true).GetOrCreate(block.ID().EpochIndex, func() *shrinkingmap.ShrinkingMap[models.BlockID, *Block] {
		return shrinkingmap.New[models.BlockID, *Block]()
	})

	return lo.Return2(attachmentsOfEpoch.GetOrCreate(block.ID(), func() *Block {
		return block
	}))
}

func (a *attachments) getEarliestAttachment(txID utxo.TransactionID) (attachment *Block) {
	var lowestTime time.Time
	if txStorage := a.storage(txID, false); txStorage != nil {
		txStorage.ForEach(func(_ epoch.Index, blocks *shrinkingmap.ShrinkingMap[models.BlockID, *Block]) bool {
			blocks.ForEach(func(_ models.BlockID, block *Block) bool {
				if lowestTime.After(block.IssuingTime()) || lowestTime.IsZero() {
					lowestTime = block.IssuingTime()
					attachment = block
				}

				return true
			})
			return true
		})
	}

	return
}

func (a *attachments) getLatestAttachment(txID utxo.TransactionID) (attachment *Block) {
	highestTime := time.Time{}
	if txStorage := a.storage(txID, false); txStorage != nil {
		txStorage.ForEach(func(_ epoch.Index, blocks *shrinkingmap.ShrinkingMap[models.BlockID, *Block]) bool {
			blocks.ForEach(func(_ models.BlockID, block *Block) bool {
				if highestTime.Before(block.IssuingTime()) {
					highestTime = block.IssuingTime()
					attachment = block
				}
				return true
			})
			return true
		})
	}

	return
}

func (a *attachments) storage(txID utxo.TransactionID, createIfMissing bool) (storage *shrinkingmap.ShrinkingMap[epoch.Index, *shrinkingmap.ShrinkingMap[models.BlockID, *Block]]) {
	if createIfMissing {
		storage, _ = a.attachments.GetOrCreate(txID, func() *shrinkingmap.ShrinkingMap[epoch.Index, *shrinkingmap.ShrinkingMap[models.BlockID, *Block]] {
			return shrinkingmap.New[epoch.Index, *shrinkingmap.ShrinkingMap[models.BlockID, *Block]]()
		})
		return
	}

	storage, _ = a.attachments.Get(txID)

	return
}

func (a *attachments) updateEvictionMap(epochIndex epoch.Index, txID utxo.TransactionID) {
	txIDs, _ := a.evictionMap.GetOrCreate(epochIndex, func() set.Set[utxo.TransactionID] {
		return set.New[utxo.TransactionID](true)
	})
	txIDs.Add(txID)
}
