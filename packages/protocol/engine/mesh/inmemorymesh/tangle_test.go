package inmemorymesh_test

import (
	"testing"
	"time"

	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool/realitiesledger"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/testmesh"
	"github.com/izuc/zipp/packages/protocol/models"
)

func Test(t *testing.T) {
	workers := workerpool.NewGroup(t.Name())
	tf := testmesh.NewDefaultTestFramework(t,
		workers.CreateGroup("LedgerTestFramework"),
		realitiesledger.NewTestLedger(t, workers.CreateGroup("Ledger")),
		slot.NewTimeProvider(time.Now().Unix(), 10),
	)

	tf.BlockDAG.CreateBlock("block1")
	tf.BlockDAG.CreateBlock("block2", models.WithStrongParents(tf.BlockDAG.BlockIDs("block1")))
	tf.BlockDAG.IssueBlocks("block1", "block2")

	tf.BlockDAG.AssertSolid(map[string]bool{
		"block1": true,
		"block2": true,
	})
}
