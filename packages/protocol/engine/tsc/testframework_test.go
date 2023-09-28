package tsc_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/blockgadget"
	"github.com/izuc/zipp/packages/protocol/engine/mesh"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
	"github.com/izuc/zipp/packages/protocol/engine/tsc"
)

// region TestFramework //////////////////////////////////////////////////////////////////////////////////////////////////////

type TestFramework struct {
	test           *testing.T
	Manager        *tsc.Manager
	MockAcceptance *blockgadget.MockBlockGadget

	Mesh          *mesh.TestFramework
	BlockDAG      *blockdag.TestFramework
	Booker        *booker.TestFramework
	VirtualVoting *booker.VirtualVotingTestFramework
}

func NewTestFramework(test *testing.T, meshTF *mesh.TestFramework, optsTSCManager ...options.Option[tsc.Manager]) *TestFramework {
	t := &TestFramework{
		test:           test,
		Mesh:           meshTF,
		BlockDAG:       meshTF.BlockDAG,
		Booker:         meshTF.Booker,
		VirtualVoting:  meshTF.VirtualVoting,
		MockAcceptance: blockgadget.NewMockAcceptanceGadget(),
	}

	t.Manager = tsc.New(t.MockAcceptance.IsBlockAccepted, meshTF.Instance, optsTSCManager...)
	t.Mesh.Booker.Instance.Events().BlockBooked.Hook(func(event *booker.BlockBookedEvent) {
		t.Manager.AddBlock(event.Block)
	})

	return t
}

func (t *TestFramework) AssertOrphaned(expectedState map[string]bool) {
	for alias, expectedOrphanage := range expectedState {
		t.Mesh.Booker.AssertBlock(alias, func(block *booker.Block) {
			require.Equal(t.test, expectedOrphanage, block.Block.IsOrphaned(), "block %s is incorrectly orphaned", block.ID())
		})
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
