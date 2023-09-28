package realitiesledger

import (
	"testing"

	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/vm/mockedvm"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
)

func NewTestLedger(t *testing.T, workers *workerpool.Group, optsLedger ...options.Option[RealitiesLedger]) mempool.MemPool {
	storage := blockdag.NewTestStorage(t, workers)
	l := New(append([]options.Option[RealitiesLedger]{
		WithVM(new(mockedvm.MockedVM)),
	}, optsLedger...)...)
	l.Initialize(workers.CreatePool("RealitiesLedger", 2), storage)

	t.Cleanup(func() {
		workers.WaitChildren()
		l.Shutdown()
		storage.Shutdown()
	})

	return l
}

func NewDefaultTestFramework(t *testing.T, workers *workerpool.Group, optsLedger ...options.Option[RealitiesLedger]) *mempool.TestFramework {
	return mempool.NewTestFramework(t, NewTestLedger(t, workers.CreateGroup("RealitiesLedger"), optsLedger...))
}
