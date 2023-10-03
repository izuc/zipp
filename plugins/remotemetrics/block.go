package remotemetrics

import (
	"time"

	"github.com/izuc/zipp.foundation/core/identity"

	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/ledger/utxo"

	"github.com/izuc/zipp/packages/core/mesh_old"

	"github.com/izuc/zipp/packages/app/remotemetrics"
)

func sendBlockSchedulerRecord(blockID mesh_old.BlockID, recordType string) {
	if !deps.Mesh.Synced() {
		return
	}
	var nodeID string
	if deps.Local != nil {
		nodeID = deps.Local.Identity.ID().String()
	}

	record := &remotemetrics.BlockScheduledMetrics{
		Type:         recordType,
		NodeID:       nodeID,
		MetricsLevel: Parameters.MetricsLevel,
		BlockID:      blockID.Base58(),
	}

	deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
		issuerID := identity.NewID(block.IssuerPublicKey())
		record.IssuedTimestamp = block.IssuingTime()
		record.IssuerID = issuerID.String()
		record.AccessMana = deps.Mesh.Scheduler.GetManaFromCache(issuerID)
		record.StrongEdgeCount = len(block.ParentsByType(mesh_old.StrongParentType))
		if weakParentsCount := len(block.ParentsByType(mesh_old.WeakParentType)); weakParentsCount > 0 {
			record.StrongEdgeCount = weakParentsCount
		}
		if likeParentsCount := len(block.ParentsByType(mesh_old.ShallowLikeParentType)); likeParentsCount > 0 {
			record.StrongEdgeCount = len(block.ParentsByType(mesh_old.ShallowLikeParentType))
		}

		deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
			record.ReceivedTimestamp = blockMetadata.ReceivedTime()
			record.ScheduledTimestamp = blockMetadata.ScheduledTime()
			record.DroppedTimestamp = blockMetadata.DiscardedTime()
			record.BookedTimestamp = blockMetadata.BookedTime()
			// may be overridden by tx data
			record.SolidTimestamp = blockMetadata.SolidificationTime()
			record.DeltaSolid = blockMetadata.SolidificationTime().Sub(record.IssuedTimestamp).Nanoseconds()
			record.QueuedTimestamp = blockMetadata.QueuedTime()
			record.DeltaBooked = blockMetadata.BookedTime().Sub(record.IssuedTimestamp).Nanoseconds()
			record.ConfirmationState = uint8(blockMetadata.ConfirmationState())
			record.ConfirmationStateTimestamp = blockMetadata.ConfirmationStateTime()
			if !blockMetadata.ConfirmationStateTime().IsZero() {
				record.DeltaConfirmationStateTime = blockMetadata.ConfirmationStateTime().Sub(record.IssuedTimestamp).Nanoseconds()
			}

			var scheduleDoneTime time.Time
			// one of those conditions must be true
			if !record.ScheduledTimestamp.IsZero() {
				scheduleDoneTime = record.ScheduledTimestamp
			} else if !record.DroppedTimestamp.IsZero() {
				scheduleDoneTime = record.DroppedTimestamp
			}
			record.DeltaScheduledIssued = scheduleDoneTime.Sub(record.IssuedTimestamp).Nanoseconds()
			record.DeltaScheduledReceived = scheduleDoneTime.Sub(blockMetadata.ReceivedTime()).Nanoseconds()
			record.DeltaReceivedIssued = blockMetadata.ReceivedTime().Sub(record.IssuedTimestamp).Nanoseconds()
			record.SchedulingTime = scheduleDoneTime.Sub(blockMetadata.QueuedTime()).Nanoseconds()
		})
	})

	// override block solidification data if block contains a transaction
	deps.Mesh.Utils.ComputeIfTransaction(blockID, func(transactionID utxo.TransactionID) {
		deps.Mesh.Ledger.Storage.CachedTransactionMetadata(transactionID).Consume(func(transactionMetadata *ledger.TransactionMetadata) {
			record.SolidTimestamp = transactionMetadata.BookingTime()
			record.TransactionID = transactionID.Base58()
			record.DeltaSolid = transactionMetadata.BookingTime().Sub(record.IssuedTimestamp).Nanoseconds()
		})
	})

	_ = deps.RemoteLogger.Send(record)
}

func onTransactionConfirmed(transactionID utxo.TransactionID) {
	if !deps.Mesh.Synced() {
		return
	}

	blockIDs := deps.Mesh.Storage.AttachmentBlockIDs(transactionID)
	if len(blockIDs) == 0 {
		return
	}

	deps.Mesh.Storage.Block(blockIDs.First()).Consume(onBlockFinalized)
}

func onBlockFinalized(block *mesh_old.Block) {
	if !deps.Mesh.Synced() {
		return
	}

	blockID := block.ID()

	var nodeID string
	if deps.Local != nil {
		nodeID = deps.Local.Identity.ID().String()
	}

	record := &remotemetrics.BlockFinalizedMetrics{
		Type:         "blockFinalized",
		NodeID:       nodeID,
		MetricsLevel: Parameters.MetricsLevel,
		BlockID:      blockID.Base58(),
	}

	issuerID := identity.NewID(block.IssuerPublicKey())
	record.IssuedTimestamp = block.IssuingTime()
	record.IssuerID = issuerID.String()
	record.StrongEdgeCount = len(block.ParentsByType(mesh_old.StrongParentType))
	if weakParentsCount := len(block.ParentsByType(mesh_old.WeakParentType)); weakParentsCount > 0 {
		record.WeakEdgeCount = weakParentsCount
	}
	if shallowLikeParentsCount := len(block.ParentsByType(mesh_old.ShallowLikeParentType)); shallowLikeParentsCount > 0 {
		record.ShallowLikeEdgeCount = shallowLikeParentsCount
	}
	deps.Mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
		record.ScheduledTimestamp = blockMetadata.ScheduledTime()
		record.DeltaScheduled = blockMetadata.ScheduledTime().Sub(record.IssuedTimestamp).Nanoseconds()
		record.BookedTimestamp = blockMetadata.BookedTime()
		record.DeltaBooked = blockMetadata.BookedTime().Sub(record.IssuedTimestamp).Nanoseconds()
	})

	deps.Mesh.Utils.ComputeIfTransaction(blockID, func(transactionID utxo.TransactionID) {
		deps.Mesh.Ledger.Storage.CachedTransactionMetadata(transactionID).Consume(func(transactionMetadata *ledger.TransactionMetadata) {
			record.SolidTimestamp = transactionMetadata.BookingTime()
			record.TransactionID = transactionID.Base58()
			record.DeltaSolid = transactionMetadata.BookingTime().Sub(record.IssuedTimestamp).Nanoseconds()
		})
	})

	_ = deps.RemoteLogger.Send(record)
}

func sendMissingBlockRecord(blockID mesh_old.BlockID, recordType string) {
	if !deps.Mesh.Synced() {
		return
	}

	var nodeID string
	if deps.Local != nil {
		nodeID = deps.Local.Identity.ID().String()
	}

	record := &remotemetrics.MissingBlockMetrics{
		Type:         recordType,
		NodeID:       nodeID,
		MetricsLevel: Parameters.MetricsLevel,
		BlockID:      blockID.Base58(),
	}

	deps.Mesh.Storage.Block(blockID).Consume(func(block *mesh_old.Block) {
		record.IssuerID = identity.NewID(block.IssuerPublicKey()).String()
	})

	_ = deps.RemoteLogger.Send(record)
}
