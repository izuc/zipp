package conflictdag

import (
	"sync"

	"github.com/iotaledger/hive.go/byteutils"
	"github.com/iotaledger/hive.go/generics/set"
	"github.com/iotaledger/hive.go/generics/walker"
	"github.com/iotaledger/hive.go/types/confirmation"
)

// ConflictDAG represents a generic DAG that is able to model causal dependencies between conflicts that try to access a
// shared set of resources.
type ConflictDAG[ConflictIDType, ResourceIDType comparable] struct {
	// Events is a dictionary for events emitted by the ConflictDAG.
	Events *Events[ConflictIDType, ResourceIDType]

	// Storage is a dictionary for storage related API endpoints.
	Storage *Storage[ConflictIDType, ResourceIDType]

	// Utils is a dictionary for utility methods that simplify the interaction with the ConflictDAG.
	Utils *Utils[ConflictIDType, ResourceIDType]

	// options is a dictionary for configuration parameters of the ConflictDAG.
	options *options

	// RWMutex is a mutex that prevents that two processes simultaneously update the ConfirmationState.
	sync.RWMutex
}

// New returns a new ConflictDAG with the given options.
func New[ConflictIDType, ResourceIDType comparable](options ...Option) (new *ConflictDAG[ConflictIDType, ResourceIDType]) {
	new = &ConflictDAG[ConflictIDType, ResourceIDType]{
		Events:  newEvents[ConflictIDType, ResourceIDType](),
		options: newOptions(options...),
	}
	new.Storage = newStorage[ConflictIDType, ResourceIDType](new.options)
	new.Utils = newUtils(new)

	return
}

// CreateConflict creates a new Conflict in the ConflictDAG and returns true if the Conflict was new.
func (b *ConflictDAG[ConflictIDType, ResourceIDType]) CreateConflict(id ConflictIDType, parents *set.AdvancedSet[ConflictIDType], conflictingResources *set.AdvancedSet[ResourceIDType]) (created bool) {
	b.RLock()
	b.Storage.CachedConflict(id, func(ConflictIDType) (conflict *Conflict[ConflictIDType, ResourceIDType]) {
		conflict = NewConflict(id, parents, set.NewAdvancedSet[ResourceIDType]())

		b.addConflictMembers(conflict, conflictingResources)
		b.createChildBranchReferences(parents, id)

		if b.anyParentRejected(conflict) || b.anyConflictingBranchAccepted(conflict) {
			conflict.setConfirmationState(confirmation.Rejected)
		}

		created = true

		return conflict
	}).Release()
	b.RUnlock()

	if created {
		b.Events.ConflictCreated.Trigger(&ConflictCreatedEvent[ConflictIDType, ResourceIDType]{
			ID:                     id,
			ParentConflictIDs:      parents,
			ConflictingResourceIDs: conflictingResources,
		})
	}

	return created
}

// UpdateConflictParents changes the parents of a Conflict after a fork (also updating the corresponding references).
func (b *ConflictDAG[ConflictIDType, ResourceIDType]) UpdateConflictParents(id ConflictIDType, removedBranchIDs *set.AdvancedSet[ConflictIDType], addedBranchID ConflictIDType) (updated bool) {
	b.RLock()

	var parentBranchIDs *set.AdvancedSet[ConflictIDType]
	b.Storage.CachedConflict(id).Consume(func(branch *Conflict[ConflictIDType, ResourceIDType]) {
		parentBranchIDs = branch.Parents()
		if !parentBranchIDs.Add(addedBranchID) {
			return
		}

		b.removeChildBranchReferences(parentBranchIDs.DeleteAll(removedBranchIDs), id)
		b.createChildBranchReferences(set.NewAdvancedSet(addedBranchID), id)

		branch.SetParents(parentBranchIDs)
		updated = true
	})
	b.RUnlock()

	if updated {
		b.Events.BranchParentsUpdated.Trigger(&BranchParentsUpdatedEvent[ConflictIDType, ResourceIDType]{
			BranchID:         id,
			AddedBranch:      addedBranchID,
			RemovedBranches:  removedBranchIDs,
			ParentsBranchIDs: parentBranchIDs,
		})
	}

	return updated
}

// UpdateConflictingResources adds the Conflict to the named conflicts - it returns true if the conflict membership was modified
// during this operation.
func (b *ConflictDAG[ConflictIDType, ResourceIDType]) UpdateConflictingResources(id ConflictIDType, conflictingResourceIDs *set.AdvancedSet[ResourceIDType]) (updated bool) {
	b.RLock()
	b.Storage.CachedConflict(id).Consume(func(branch *Conflict[ConflictIDType, ResourceIDType]) {
		updated = b.addConflictMembers(branch, conflictingResourceIDs)
	})
	b.RUnlock()

	if updated {
		b.Events.BranchConflictsUpdated.Trigger(&BranchConflictsUpdatedEvent[ConflictIDType, ResourceIDType]{
			BranchID:       id,
			NewConflictIDs: conflictingResourceIDs,
		})
	}

	return updated
}

// UnconfirmedConflicts takes a set of BranchIDs and removes all the Accepted/Confirmed Branches (leaving only the
// pending or rejected ones behind).
func (b *ConflictDAG[ConflictIDType, ConflictingResourceID]) UnconfirmedConflicts(branchIDs *set.AdvancedSet[ConflictIDType]) (pendingBranchIDs *set.AdvancedSet[ConflictIDType]) {
	if !b.options.mergeToMaster {
		return branchIDs.Clone()
	}

	pendingBranchIDs = set.NewAdvancedSet[ConflictIDType]()
	for branchWalker := branchIDs.Iterator(); branchWalker.HasNext(); {
		if currentBranchID := branchWalker.Next(); !b.confirmationState(currentBranchID).IsAccepted() {
			pendingBranchIDs.Add(currentBranchID)
		}
	}

	return pendingBranchIDs
}

// SetBranchAccepted sets the ConfirmationState of the given Conflict to be Accepted - it automatically sets also the
// conflicting branches to be rejected.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) SetBranchAccepted(branchID ConflictID) (modified bool) {
	b.Lock()
	defer b.Unlock()

	rejectionWalker := walker.New[ConflictID]()
	for confirmationWalker := set.NewAdvancedSet(branchID).Iterator(); confirmationWalker.HasNext(); {
		b.Storage.CachedConflict(confirmationWalker.Next()).Consume(func(branch *Conflict[ConflictID, ConflictingResourceID]) {
			if modified = branch.setConfirmationState(confirmation.Accepted); !modified {
				return
			}

			b.Events.BranchAccepted.Trigger(&BranchAcceptedEvent[ConflictID]{
				ID: branchID,
			})

			confirmationWalker.PushAll(branch.Parents().Slice()...)

			b.Utils.forEachConflictingBranchID(branch, func(conflictingBranchID ConflictID) bool {
				rejectionWalker.Push(conflictingBranchID)
				return true
			})
		})
	}

	for rejectionWalker.HasNext() {
		b.Storage.CachedConflict(rejectionWalker.Next()).Consume(func(branch *Conflict[ConflictID, ConflictingResourceID]) {
			if modified = branch.setConfirmationState(confirmation.Rejected); !modified {
				return
			}

			b.Events.BranchRejected.Trigger(&BranchRejectedEvent[ConflictID]{
				ID: branch.ID(),
			})

			b.Storage.CachedChildBranches(branch.ID()).Consume(func(childBranch *ChildBranch[ConflictID]) {
				rejectionWalker.Push(childBranch.ChildBranchID())
			})
		})
	}

	return modified
}

// ConfirmationState returns the ConfirmationState of the given BranchIDs.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) ConfirmationState(branchIDs *set.AdvancedSet[ConflictID]) (confirmationState confirmation.State) {
	b.RLock()
	defer b.RUnlock()

	confirmationState = confirmation.Confirmed
	for it := branchIDs.Iterator(); it.HasNext(); {
		if confirmationState = confirmationState.Aggregate(b.confirmationState(it.Next())); confirmationState.IsRejected() {
			return confirmation.Rejected
		}
	}

	return confirmationState
}

// Shutdown shuts down the stateful elements of the ConflictDAG (the Storage).
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) Shutdown() {
	b.Storage.Shutdown()
}

// addConflictMembers creates the named ConflictMember references.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) addConflictMembers(branch *Conflict[ConflictID, ConflictingResourceID], conflictIDs *set.AdvancedSet[ConflictingResourceID]) (added bool) {
	for it := conflictIDs.Iterator(); it.HasNext(); {
		conflictID := it.Next()

		if added = branch.addConflict(conflictID); added {
			b.registerConflictMember(conflictID, branch.ID())
		}
	}

	return added
}

// createChildBranchReferences creates the named ChildBranch references.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) createChildBranchReferences(parentBranchIDs *set.AdvancedSet[ConflictID], childBranchID ConflictID) {
	for it := parentBranchIDs.Iterator(); it.HasNext(); {
		b.Storage.CachedChildBranch(it.Next(), childBranchID, NewChildBranch[ConflictID]).Release()
	}
}

// removeChildBranchReferences removes the named ChildBranch references.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) removeChildBranchReferences(parentBranchIDs *set.AdvancedSet[ConflictID], childBranchID ConflictID) {
	for it := parentBranchIDs.Iterator(); it.HasNext(); {
		b.Storage.childBranchStorage.Delete(byteutils.ConcatBytes(bytes(it.Next()), bytes(childBranchID)))
	}
}

// anyParentRejected checks if any of a Branches parents is Rejected.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) anyParentRejected(branch *Conflict[ConflictID, ConflictingResourceID]) (rejected bool) {
	for it := branch.Parents().Iterator(); it.HasNext(); {
		if b.confirmationState(it.Next()).IsRejected() {
			return true
		}
	}

	return false
}

// anyConflictingBranchAccepted checks if any conflicting Conflict is Accepted/Confirmed.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) anyConflictingBranchAccepted(branch *Conflict[ConflictID, ConflictingResourceID]) (anyConfirmed bool) {
	b.Utils.forEachConflictingBranchID(branch, func(conflictingBranchID ConflictID) bool {
		anyConfirmed = b.confirmationState(conflictingBranchID).IsAccepted()
		return !anyConfirmed
	})

	return anyConfirmed
}

// registerConflictMember registers a Conflict in a Conflict by creating the references (if necessary) and increasing the
// corresponding member counter.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) registerConflictMember(conflictID ConflictingResourceID, branchID ConflictID) {
	b.Storage.CachedConflictMember(conflictID, branchID, NewConflictMember[ConflictingResourceID, ConflictID]).Release()
}

// confirmationState returns the ConfirmationState of the Conflict with the given ConflictID.
func (b *ConflictDAG[ConflictID, ConflictingResourceID]) confirmationState(branchID ConflictID) (confirmationState confirmation.State) {
	b.Storage.CachedConflict(branchID).Consume(func(branch *Conflict[ConflictID, ConflictingResourceID]) {
		confirmationState = branch.ConfirmationState()
	})

	return confirmationState
}
