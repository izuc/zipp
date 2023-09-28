package inmemoryblockdag

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/core/commitment"
	"github.com/izuc/zipp/packages/protocol/engine/eviction"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
)

func NewTestBlockDAG(t *testing.T, workers *workerpool.Group, evictionState *eviction.State, slotTimeProvider *slot.TimeProvider, commitmentLoadFunc func(index slot.Index) (commitment *commitment.Commitment, err error), optsBlockDAG ...options.Option[BlockDAG]) *BlockDAG {
	require.NotNil(t, evictionState)
	return New(workers, evictionState, func() *slot.TimeProvider { return slotTimeProvider }, commitmentLoadFunc, optsBlockDAG...)
}

func NewDefaultTestFramework(t *testing.T, workers *workerpool.Group, optsBlockDAG ...options.Option[BlockDAG]) *blockdag.TestFramework {
	storageInstance := blockdag.NewTestStorage(t, workers)
	b := NewTestBlockDAG(t, workers.CreateGroup("BlockDAG"), eviction.NewState(storageInstance), slot.NewTimeProvider(time.Now().Unix(), 10), blockdag.DefaultCommitmentFunc, optsBlockDAG...)

	return blockdag.NewTestFramework(t,
		workers.CreateGroup("BlockDAGTestFramework"),
		b,
		b.slotTimeProviderFunc,
	)
}
