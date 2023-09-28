package ledger

import (
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool"
)

// StateDiffs is a submodule that provides access to the state diffs of the ledger state.
type StateDiffs interface {
	// StreamCreatedOutputs streams the created outputs of the given slot index.
	StreamCreatedOutputs(slot.Index, func(*mempool.OutputWithMetadata) error) error

	// StreamSpentOutputs streams the spent outputs of the given slot index.
	StreamSpentOutputs(slot.Index, func(*mempool.OutputWithMetadata) error) error
}
