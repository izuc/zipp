package mesh_old

import (
	"time"

	"github.com/cockroachdb/errors"
	"github.com/izuc/zipp.foundation/core/generics/set"
	"github.com/izuc/zipp.foundation/core/generics/walker"

	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
)

// region utils ////////////////////////////////////////////////////////////////////////////////////////////////////////

// Utils is a Mesh component that bundles methods that can be used to interact with the Mesh, that do not belong
// into public API.
type Utils struct {
	mesh *Mesh
}

// NewUtils is the constructor of the Utils component.
func NewUtils(mesh *Mesh) (utils *Utils) {
	return &Utils{
		mesh: mesh,
	}
}

// region walkers //////////////////////////////////////////////////////////////////////////////////////////////////////

// WalkBlockID is a generic Mesh walker that executes a custom callback for every visited BlockID, starting from
// the given entry points. It accepts an optional boolean parameter which can be set to true if a Block should be
// visited more than once following different paths. The callback receives a Walker object as the last parameter which
// can be used to control the behavior of the walk similar to how a "Context" is used in some parts of the stdlib.
func (u *Utils) WalkBlockID(callback func(blockID BlockID, walker *walker.Walker[BlockID]), entryPoints BlockIDs, revisitElements ...bool) {
	if len(entryPoints) == 0 {
		return
	}

	blockIDWalker := walker.New[BlockID](revisitElements...)
	for blockID := range entryPoints {
		blockIDWalker.Push(blockID)
	}

	for blockIDWalker.HasNext() {
		callback(blockIDWalker.Next(), blockIDWalker)
	}
}

// WalkBlock is a generic Mesh walker that executes a custom callback for every visited Block, starting from
// the given entry points. It accepts an optional boolean parameter which can be set to true if a Block should be
// visited more than once following different paths. The callback receives a Walker object as the last parameter which
// can be used to control the behavior of the walk similar to how a "Context" is used in some parts of the stdlib.
func (u *Utils) WalkBlock(callback func(block *Block, walker *walker.Walker[BlockID]), entryPoints BlockIDs, revisitElements ...bool) {
	u.WalkBlockID(func(blockID BlockID, walker *walker.Walker[BlockID]) {
		u.mesh.Storage.Block(blockID).Consume(func(block *Block) {
			callback(block, walker)
		})
	}, entryPoints, revisitElements...)
}

// WalkBlockMetadata is a generic Mesh walker that executes a custom callback for every visited BlockMetadata,
// starting from the given entry points. It accepts an optional boolean parameter which can be set to true if a Block
// should be visited more than once following different paths. The callback receives a Walker object as the last
// parameter which can be used to control the behavior of the walk similar to how a "Context" is used in some parts of
// the stdlib.
func (u *Utils) WalkBlockMetadata(callback func(blockMetadata *BlockMetadata, walker *walker.Walker[BlockID]), entryPoints BlockIDs, revisitElements ...bool) {
	u.WalkBlockID(func(blockID BlockID, walker *walker.Walker[BlockID]) {
		u.mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *BlockMetadata) {
			callback(blockMetadata, walker)
		})
	}, entryPoints, revisitElements...)
}

// WalkBlockAndMetadata is a generic Mesh walker that executes a custom callback for every visited Block and
// BlockMetadata, starting from the given entry points. It accepts an optional boolean parameter which can be set to
// true if a Block should be visited more than once following different paths. The callback receives a Walker object
// as the last parameter which can be used to control the behavior of the walk similar to how a "Context" is used in
// some parts of the stdlib.
func (u *Utils) WalkBlockAndMetadata(callback func(block *Block, blockMetadata *BlockMetadata, walker *walker.Walker[BlockID]), entryPoints BlockIDs, revisitElements ...bool) {
	u.WalkBlockID(func(blockID BlockID, walker *walker.Walker[BlockID]) {
		u.mesh.Storage.Block(blockID).Consume(func(block *Block) {
			u.mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *BlockMetadata) {
				callback(block, blockMetadata, walker)
			})
		})
	}, entryPoints, revisitElements...)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region structural checks ////////////////////////////////////////////////////////////////////////////////////////////
// checkBookedParents check if block parents are booked and add then to bookedParents. If we find attachmentBlockId in the parents we stop and return true.
func (u *Utils) checkBookedParents(block *Block, attachmentBlockID BlockID, getParents func(*Block) BlockIDs) (bool, BlockIDs) {
	bookedParents := NewBlockIDs()

	for parentID := range getParents(block) {
		var parentBooked bool
		u.mesh.Storage.BlockMetadata(parentID).Consume(func(parentMetadata *BlockMetadata) {
			parentBooked = parentMetadata.IsBooked()
		})
		if !parentBooked {
			continue
		}

		// First check all of the parents to avoid unnecessary checks and possible walking.
		if attachmentBlockID == parentID {
			return true, bookedParents
		}

		bookedParents.Add(parentID)
	}
	return false, bookedParents
}

// ApprovingBlockIDs returns the BlockIDs that approve a given Block. It accepts an optional ChildType to
// filter the Children.
func (u *Utils) ApprovingBlockIDs(blockID BlockID, optionalChildType ...ChildType) (approvingBlockIDs BlockIDs) {
	approvingBlockIDs = NewBlockIDs()
	u.mesh.Storage.Children(blockID, optionalChildType...).Consume(func(child *Child) {
		approvingBlockIDs.Add(child.ChildBlockID())
	})

	return
}

// AllConflictsLiked returns true if all the passed conflicts are liked.
func (u *Utils) AllConflictsLiked(conflictIDs *set.AdvancedSet[utxo.TransactionID]) bool {
	for it := conflictIDs.Iterator(); it.HasNext(); {
		if !u.mesh.OMVConsensusManager.ConflictLiked(it.Next()) {
			return false
		}
	}

	return true
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// ComputeIfTransaction computes the given callback if the given blockID contains a transaction.
func (u *Utils) ComputeIfTransaction(blockID BlockID, compute func(utxo.TransactionID)) (computed bool) {
	u.mesh.Storage.Block(blockID).Consume(func(block *Block) {
		if tx, ok := block.Payload().(utxo.Transaction); ok {
			transactionID := tx.ID()
			compute(transactionID)
			computed = true
		}
	})
	return
}

// FirstAttachment returns the BlockID and timestamp of the first (oldest) attachment of a given transaction.
func (u *Utils) FirstAttachment(transactionID utxo.TransactionID) (oldestAttachmentTime time.Time, oldestAttachmentBlockID BlockID, err error) {
	oldestAttachmentTime = time.Unix(0, 0)
	oldestAttachmentBlockID = EmptyBlockID
	if !u.mesh.Storage.Attachments(transactionID).Consume(func(attachment *Attachment) {
		u.mesh.Storage.Block(attachment.BlockID()).Consume(func(block *Block) {
			if oldestAttachmentTime.Unix() == 0 || block.IssuingTime().Before(oldestAttachmentTime) {
				oldestAttachmentTime = block.IssuingTime()
				oldestAttachmentBlockID = block.ID()
			}
		})
	}) {
		err = errors.Errorf("could not find any attachments of transaction: %s", transactionID.String())
	}
	return
}

// ConfirmedConsumer returns the confirmed transactionID consuming the given outputID.
func (u *Utils) ConfirmedConsumer(outputID utxo.OutputID) (consumerID utxo.TransactionID) {
	// default to no consumer, i.e. Genesis
	consumerID = utxo.EmptyTransactionID
	u.mesh.Ledger.Storage.CachedConsumers(outputID).Consume(func(consumer *ledger.Consumer) {
		if consumerID != utxo.EmptyTransactionID {
			return
		}
		if u.mesh.ConfirmationOracle.IsTransactionConfirmed(consumer.TransactionID()) {
			consumerID = consumer.TransactionID()
		}
	})
	return
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
