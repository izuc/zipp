package mesh

import (
	"testing"

	"github.com/izuc/zipp/packages/core/votes"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
)

type TestFramework struct {
	test     *testing.T
	Instance Mesh

	VirtualVoting *booker.VirtualVotingTestFramework
	Booker        *booker.TestFramework
	MemPool       *mempool.TestFramework
	BlockDAG      *blockdag.TestFramework
	Votes         *votes.TestFramework
}

func NewTestFramework(test *testing.T, mesh Mesh, bookerTF *booker.TestFramework) *TestFramework {
	return &TestFramework{
		test:          test,
		Instance:      mesh,
		Booker:        bookerTF,
		VirtualVoting: bookerTF.VirtualVoting,
		MemPool:       bookerTF.Ledger,
		BlockDAG:      bookerTF.BlockDAG,
		Votes:         bookerTF.VirtualVoting.Votes,
	}
}
