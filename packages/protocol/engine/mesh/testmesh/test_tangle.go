package testmesh

import (
	"testing"

	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/kvstore/mapdb"
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/protocol/engine/eviction"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool"
	"github.com/izuc/zipp/packages/protocol/engine/mesh"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag/inmemoryblockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker/markerbooker"
	"github.com/izuc/zipp/packages/protocol/engine/sybilprotection"
	"github.com/izuc/zipp/packages/protocol/markers"
)

type TestMesh struct {
	events   *mesh.Events
	blockDAG *inmemoryblockdag.BlockDAG
	booker   *markerbooker.Booker

	slotTimeProvider *slot.TimeProvider
	memPool          mempool.MemPool
	evictionState    *eviction.State
	validators       *sybilprotection.WeightedSet

	module.Module
}

func NewTestMesh(t *testing.T, workers *workerpool.Group, slotTimeProvider *slot.TimeProvider, memPool mempool.MemPool, validators *sybilprotection.WeightedSet, optsBooker ...options.Option[markerbooker.Booker]) *TestMesh {
	storageInstance := blockdag.NewTestStorage(t, workers)

	testMesh := &TestMesh{
		events:           mesh.NewEvents(),
		slotTimeProvider: slotTimeProvider,
		memPool:          memPool,
		evictionState:    eviction.NewState(storageInstance),
		validators:       validators,
	}

	testMesh.blockDAG = inmemoryblockdag.New(workers.CreateGroup("BlockDAG"), testMesh.evictionState, testMesh.SlotTimeProvider, storageInstance.Commitments.Load)
	testMesh.booker = markerbooker.New(workers.CreateGroup("Booker"), testMesh.evictionState, memPool, validators, testMesh.SlotTimeProvider,
		append([]options.Option[markerbooker.Booker]{
			markerbooker.WithSlotCutoffCallback(func() slot.Index {
				return 0
			}),
			markerbooker.WithSequenceCutoffCallback(func(id markers.SequenceID) markers.Index {
				return 1
			}),
		}, optsBooker...)...,
	)
	testMesh.booker.Initialize(testMesh.blockDAG)

	testMesh.TriggerConstructed()
	testMesh.TriggerInitialized()

	return testMesh
}

var _ mesh.Mesh = new(TestMesh)

func (t *TestMesh) Events() *mesh.Events {
	return t.events
}

func (t *TestMesh) Booker() booker.Booker {
	return t.booker
}

func (t *TestMesh) BlockDAG() blockdag.BlockDAG {
	return t.blockDAG
}

func (t *TestMesh) MemPool() mempool.MemPool {
	return t.memPool
}

func (t *TestMesh) EvictionState() *eviction.State {
	return t.evictionState
}

func (t *TestMesh) SlotTimeProvider() *slot.TimeProvider {
	return t.slotTimeProvider
}

func (t *TestMesh) Validators() *sybilprotection.WeightedSet {
	return t.validators
}

func NewDefaultTestFramework(t *testing.T, workers *workerpool.Group, memPool mempool.MemPool, slotTimeProvider *slot.TimeProvider, optsBooker ...options.Option[markerbooker.Booker]) *mesh.TestFramework {
	validators := sybilprotection.NewWeightedSet(sybilprotection.NewWeights(mapdb.NewMapDB()))

	testMesh := NewTestMesh(t, workers.CreateGroup("Mesh"),
		slotTimeProvider,
		memPool,
		validators,
		optsBooker...,
	)

	return mesh.NewTestFramework(t, testMesh, booker.NewTestFramework(t, workers.CreateGroup("BookerTestFramework"),
		testMesh.booker,
		testMesh.blockDAG,
		testMesh.memPool,
		validators,
		testMesh.SlotTimeProvider,
	))
}
