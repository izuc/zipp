package mesh_old

import (
	"sync"

	"github.com/cockroachdb/errors"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/generics/set"

	"github.com/izuc/zipp/packages/core/conflictdag"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
)

type TipsConflictTracker struct {
	missingConflicts  utxo.TransactionIDs
	tipsConflictCount map[utxo.TransactionID]int
	mesh            *Mesh

	sync.RWMutex
}

func NewTipsConflictTracker(mesh *Mesh) *TipsConflictTracker {
	return &TipsConflictTracker{
		missingConflicts:  set.NewAdvancedSet[utxo.TransactionID](),
		tipsConflictCount: make(map[utxo.TransactionID]int),
		mesh:            mesh,
	}
}

func (c *TipsConflictTracker) Setup() {
	c.mesh.Ledger.ConflictDAG.Events.ConflictAccepted.Attach(event.NewClosure(func(event *conflictdag.ConflictAcceptedEvent[utxo.TransactionID]) {
		c.deleteConflict(event.ID)
	}))
	c.mesh.Ledger.ConflictDAG.Events.ConflictRejected.Attach(event.NewClosure(func(event *conflictdag.ConflictRejectedEvent[utxo.TransactionID]) {
		c.deleteConflict(event.ID)
	}))
}

func (c *TipsConflictTracker) AddTip(blockID BlockID) {
	blockConflictIDs, err := c.mesh.Booker.BlockConflictIDs(blockID)
	if err != nil {
		panic(err)
	}

	c.Lock()
	defer c.Unlock()

	for it := blockConflictIDs.Iterator(); it.HasNext(); {
		conflictID := it.Next()

		if !c.mesh.Ledger.ConflictDAG.ConfirmationState(set.NewAdvancedSet(conflictID)).IsPending() {
			continue
		}

		if c.tipsConflictCount[conflictID]++; c.tipsConflictCount[conflictID] == 1 {
			c.missingConflicts.Delete(conflictID)
		}
	}
}

func (c *TipsConflictTracker) RemoveTip(blockID BlockID) {
	blockConflictIDs, err := c.mesh.Booker.BlockConflictIDs(blockID)
	if err != nil {
		panic("could not determine ConflictIDs of tip.")
	}

	c.Lock()
	defer c.Unlock()

	for it := blockConflictIDs.Iterator(); it.HasNext(); {
		conflictID := it.Next()

		if _, exists := c.tipsConflictCount[conflictID]; !exists {
			continue
		}

		if !c.mesh.Ledger.ConflictDAG.ConfirmationState(set.NewAdvancedSet(conflictID)).IsPending() {
			continue
		}

		if c.tipsConflictCount[conflictID]--; c.tipsConflictCount[conflictID] == 0 {
			c.missingConflicts.Add(conflictID)
			delete(c.tipsConflictCount, conflictID)
		}
	}
}

func (c *TipsConflictTracker) MissingConflicts(amount int) (missingConflicts utxo.TransactionIDs) {
	c.Lock()
	defer c.Unlock()

	missingConflicts = utxo.NewTransactionIDs()
	_ = c.missingConflicts.ForEach(func(conflictID utxo.TransactionID) (err error) {
		// TODO: this should not be necessary if ConflictAccepted/ConflictRejected events are fired appropriately
		if !c.mesh.Ledger.ConflictDAG.ConfirmationState(set.NewAdvancedSet(conflictID)).IsPending() {
			c.missingConflicts.Delete(conflictID)
			delete(c.tipsConflictCount, conflictID)
			return
		}
		if !c.mesh.OMVConsensusManager.ConflictLiked(conflictID) {
			return
		}

		if missingConflicts.Add(conflictID) && missingConflicts.Size() == amount {
			return errors.Errorf("amount of missing conflicts reached %d", amount)
		}

		return nil
	})

	return missingConflicts
}

func (c *TipsConflictTracker) deleteConflict(id utxo.TransactionID) {
	c.Lock()
	defer c.Unlock()

	c.missingConflicts.Delete(id)
	delete(c.tipsConflictCount, id)
}
