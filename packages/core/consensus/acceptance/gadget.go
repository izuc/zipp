package acceptance

import (
	"sync"

	"github.com/izuc/zipp.foundation/core/generics/set"
	"github.com/izuc/zipp.foundation/core/generics/walker"
	"github.com/izuc/zipp.foundation/core/types/confirmation"

	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/markers"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

// BlockThresholdTranslation is a function which translates approval weight to a confirmation.State.
type BlockThresholdTranslation func(aw float64) confirmation.State

// ConflictThresholdTranslation is a function which translates approval weight to a confirmation.State.
type ConflictThresholdTranslation func(conflictID utxo.TransactionID, aw float64) confirmation.State

const (
	acceptanceThreshold = 0.67
)

var (
	// DefaultConflictTranslation is the default function to translate the approval weight to confirmation.State of a conflict.
	DefaultConflictTranslation ConflictThresholdTranslation = func(_ utxo.TransactionID, aw float64) confirmation.State {
		if aw >= acceptanceThreshold {
			return confirmation.Accepted
		}

		return confirmation.Pending
	}

	// DefaultBlockTranslation is the default function to translate the approval weight to confirmation.State of a block.
	DefaultBlockTranslation BlockThresholdTranslation = func(aw float64) confirmation.State {
		if aw >= acceptanceThreshold {
			return confirmation.Accepted
		}

		return confirmation.Pending
	}
)

// Option is a function setting an option on an Options struct.
type Option func(*Options)

// Options defines the options for a Gadget.
type Options struct {
	ConflictTransFunc ConflictThresholdTranslation
	BlockTransFunc    BlockThresholdTranslation
}

var defaultOpts = []Option{
	WithConflictThresholdTranslation(DefaultConflictTranslation),
	WithBlockThresholdTranslation(DefaultBlockTranslation),
}

// WithBlockThresholdTranslation returns an Option setting the BlockThresholdTranslation.
func WithBlockThresholdTranslation(f BlockThresholdTranslation) Option {
	return func(opts *Options) {
		opts.BlockTransFunc = f
	}
}

// WithConflictThresholdTranslation returns an Option setting the ConflictThresholdTranslation.
func WithConflictThresholdTranslation(f ConflictThresholdTranslation) Option {
	return func(opts *Options) {
		opts.ConflictTransFunc = f
	}
}

// Gadget is a GadgetInterface which simply translates approval weight down to confirmation.State
// and then applies it to blocks, conflicts, transactions and outputs.
type Gadget struct {
	mesh                    *mesh_old.Mesh
	opts                      *Options
	lastConfirmedMarkers      map[markers.SequenceID]markers.Index
	lastConfirmedMarkersMutex sync.RWMutex
	events                    *mesh_old.ConfirmationEvents
}

// NewSimpleFinalityGadget creates a new Gadget.
func NewSimpleFinalityGadget(t *mesh_old.Mesh, opts ...Option) *Gadget {
	sfg := &Gadget{
		mesh:               t,
		opts:                 &Options{},
		lastConfirmedMarkers: make(map[markers.SequenceID]markers.Index),
		events:               mesh_old.NewConfirmationEvents(),
	}

	for _, defOpt := range defaultOpts {
		defOpt(sfg.opts)
	}
	for _, opt := range opts {
		opt(sfg.opts)
	}

	return sfg
}

// Events returns the events this gadget exposes.
func (s *Gadget) Events() *mesh_old.ConfirmationEvents {
	return s.events
}

// IsMarkerConfirmed returns whether the given marker is confirmed.
func (s *Gadget) IsMarkerConfirmed(marker markers.Marker) (confirmed bool) {
	blockID := s.mesh.Booker.MarkersManager.BlockID(marker)
	if blockID == mesh_old.EmptyBlockID {
		return false
	}

	s.mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
		if blockMetadata.ConfirmationState().IsAccepted() {
			confirmed = true
		}
	})
	return
}

// IsBlockConfirmed returns whether the given block is confirmed.
func (s *Gadget) IsBlockConfirmed(blkID mesh_old.BlockID) (confirmed bool) {
	s.mesh.Storage.BlockMetadata(blkID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
		if blockMetadata.ConfirmationState().IsAccepted() {
			confirmed = true
		}
	})
	return
}

// FirstUnconfirmedMarkerIndex returns the first Index in the given Sequence that was not confirmed, yet.
func (s *Gadget) FirstUnconfirmedMarkerIndex(sequenceID markers.SequenceID) (index markers.Index) {
	s.lastConfirmedMarkersMutex.Lock()
	defer s.lastConfirmedMarkersMutex.Unlock()

	// TODO: MAP GROWS INDEFINITELY
	index, exists := s.lastConfirmedMarkers[sequenceID]
	if exists {
		return index + 1
	}

	s.mesh.Booker.MarkersManager.Manager.Sequence(sequenceID).Consume(func(sequence *markers.Sequence) {
		index = sequence.LowestIndex()
	})

	if !s.mesh.ConfirmationOracle.IsMarkerConfirmed(markers.NewMarker(sequenceID, index)) {
		return index
	}

	// do-while loop
	s.lastConfirmedMarkers[sequenceID] = index
	index++
	for s.mesh.ConfirmationOracle.IsMarkerConfirmed(markers.NewMarker(sequenceID, index)) {
		s.lastConfirmedMarkers[sequenceID] = index
		index++
	}

	return index
}

// IsConflictConfirmed returns whether the given conflict is confirmed.
func (s *Gadget) IsConflictConfirmed(conflictID utxo.TransactionID) (confirmed bool) {
	return s.mesh.Ledger.ConflictDAG.ConfirmationState(utxo.NewTransactionIDs(conflictID)).IsAccepted()
}

// IsTransactionConfirmed returns whether the given transaction is confirmed.
func (s *Gadget) IsTransactionConfirmed(transactionID utxo.TransactionID) (confirmed bool) {
	s.mesh.Ledger.Storage.CachedTransactionMetadata(transactionID).Consume(func(transactionMetadata *ledger.TransactionMetadata) {
		if transactionMetadata.ConfirmationState().IsAccepted() {
			confirmed = true
		}
	})
	return
}

// HandleMarker receives a marker and its current approval weight. It propagates the ConfirmationState according to AW to its past cone.
func (s *Gadget) HandleMarker(marker markers.Marker, aw float64) (err error) {
	confirmationState := s.opts.BlockTransFunc(aw)
	if confirmationState.IsPending() {
		return nil
	}

	// get block ID of marker
	blockID := s.mesh.Booker.MarkersManager.BlockID(marker)
	s.mesh.Storage.BlockMetadata(blockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
		if confirmationState <= blockMetadata.ConfirmationState() {
			return
		}

		if confirmationState.IsAccepted() {
			s.setMarkerConfirmed(marker)
		}

		s.propagateConfirmationStateToBlockPastCone(blockID, confirmationState)
	})

	return err
}

// setMarkerConfirmed marks the current Marker as confirmed.
func (s *Gadget) setMarkerConfirmed(marker markers.Marker) (updated bool) {
	s.lastConfirmedMarkersMutex.Lock()
	defer s.lastConfirmedMarkersMutex.Unlock()

	if s.lastConfirmedMarkers[marker.SequenceID()] > marker.Index() {
		return false
	}

	s.lastConfirmedMarkers[marker.SequenceID()] = marker.Index()

	return true
}

// propagateConfirmationStateToBlockPastCone propagates the given ConfirmationState to the past cone of the Block.
func (s *Gadget) propagateConfirmationStateToBlockPastCone(blockID mesh_old.BlockID, confirmationState confirmation.State) {
	strongParentWalker := walker.New[mesh_old.BlockID](false).Push(blockID)
	weakParentsSet := set.New[mesh_old.BlockID]()

	for strongParentWalker.HasNext() {
		strongParentBlockID := strongParentWalker.Next()
		if strongParentBlockID == mesh_old.EmptyBlockID {
			continue
		}

		s.mesh.Storage.BlockMetadata(strongParentBlockID).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
			if blockMetadata.ConfirmationState() >= confirmationState {
				return
			}

			s.mesh.Storage.Block(strongParentBlockID).Consume(func(block *mesh_old.Block) {
				if !s.setBlockConfirmationState(block, blockMetadata, confirmationState) {
					return
				}

				block.ForEachParent(func(parent mesh_old.Parent) {
					if parent.Type == mesh_old.StrongParentType {
						strongParentWalker.Push(parent.ID)
						return
					}
					weakParentsSet.Add(parent.ID)
				})
			})
		})
	}

	weakParentsSet.ForEach(func(weakParent mesh_old.BlockID) {
		if strongParentWalker.Pushed(weakParent) {
			return
		}
		s.mesh.Storage.BlockMetadata(weakParent).Consume(func(blockMetadata *mesh_old.BlockMetadata) {
			if blockMetadata.ConfirmationState() >= confirmationState {
				return
			}

			s.mesh.Storage.Block(weakParent).Consume(func(block *mesh_old.Block) {
				s.setBlockConfirmationState(block, blockMetadata, confirmationState)
			})
		})
	})
}

// HandleConflict receives a conflictID and its approval weight. It propagates the ConfirmationState according to AW to transactions
// in the conflict (UTXO future cone) and their outputs.
func (s *Gadget) HandleConflict(conflictID utxo.TransactionID, aw float64) (err error) {
	if s.opts.ConflictTransFunc(conflictID, aw).IsAccepted() {
		s.mesh.Ledger.ConflictDAG.SetConflictAccepted(conflictID)
	}

	return nil
}

func (s *Gadget) setBlockConfirmationState(block *mesh_old.Block, blockMetadata *mesh_old.BlockMetadata, confirmationState confirmation.State) (modified bool) {
	// abort if block has ConfirmationState already set
	if modified = blockMetadata.SetConfirmationState(confirmationState); !modified {
		return
	}

	if confirmationState.IsAccepted() {
		s.Events().BlockAccepted.Trigger(&mesh_old.BlockAcceptedEvent{
			Block: block,
		})

		// set ConfirmationState of payload (applicable only to transactions)
		if tx, ok := block.Payload().(*devnetvm.Transaction); ok {
			s.mesh.Ledger.SetTransactionInclusionTime(tx.ID(), block.IssuingTime())
		}
	}

	return modified
}
