package blockgadget

import (
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/protocol/markers"
	"github.com/izuc/zipp/packages/protocol/models"
)

type Gadget interface {
	Events() *Events

	// IsBlockAccepted returns whether the given block is accepted.
	IsBlockAccepted(blockID models.BlockID) (accepted bool)

	IsBlockConfirmed(blockID models.BlockID) bool

	// IsMarkerAccepted returns whether the given marker is accepted.
	IsMarkerAccepted(marker markers.Marker) (accepted bool)

	FirstUnacceptedIndex(sequenceID markers.SequenceID) (firstUnacceptedIndex markers.Index)

	module.Interface
}
