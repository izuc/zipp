package markerbooker

import (
	"testing"
	"time"

	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/kvstore/mapdb"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/protocol/engine/eviction"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag/inmemoryblockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
	"github.com/izuc/zipp/packages/protocol/engine/sybilprotection"
)

func NewDefaultTestFramework(t *testing.T, workers *workerpool.Group, ledger mempool.MemPool, optsBooker ...options.Option[Booker]) *booker.TestFramework {
	storageInstance := blockdag.NewTestStorage(t, workers)
	evictionState := eviction.NewState(storageInstance)
	slotTimeProvider := slot.NewTimeProvider(time.Now().Unix(), 10)
	blockDAG := inmemoryblockdag.NewTestBlockDAG(t, workers, evictionState, slotTimeProvider, blockdag.DefaultCommitmentFunc)

	validators := sybilprotection.NewWeightedSet(sybilprotection.NewWeights(mapdb.NewMapDB()))

	markerBooker := New(
		workers.CreateGroup("Booker"),
		evictionState,
		ledger,
		validators,
		blockDAG.SlotTimeProvider,
		optsBooker...,
	)
	markerBooker.Initialize(blockDAG)

	return booker.NewTestFramework(t, workers, markerBooker, blockDAG, ledger, validators, func() *slot.TimeProvider {
		return slotTimeProvider
	})
}
