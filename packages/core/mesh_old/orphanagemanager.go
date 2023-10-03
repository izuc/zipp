package mesh_old

import (
	"sync"

	"github.com/izuc/zipp.foundation/core/generics/event"
)

// region OrphanageManager /////////////////////////////////////////////////////////////////////////////////////////////

// OrphanageManager is a manager that tracks orphaned blocks.
type OrphanageManager struct {
	Events *OrphanageManagerEvents

	mesh              *Mesh
	strongChildCounters map[BlockID]int
	sync.Mutex
}

// NewOrphanageManager returns a new OrphanageManager.
func NewOrphanageManager(mesh *Mesh) *OrphanageManager {
	return &OrphanageManager{
		Events: &OrphanageManagerEvents{
			BlockOrphaned:       event.New[*BlockOrphanedEvent](),
			AllChildrenOrphaned: event.New[*Block](),
		},

		mesh:              mesh,
		strongChildCounters: make(map[BlockID]int),
	}
}

func (o *OrphanageManager) Setup() {
	o.mesh.Solidifier.Events.BlockSolid.Attach(event.NewClosure(func(event *BlockSolidEvent) {
		for strongParent := range event.Block.ParentsByType(StrongParentType) {
			o.increaseStrongChildCounter(strongParent)
		}
	}))

	o.mesh.ConfirmationOracle.Events().BlockAccepted.Attach(event.NewClosure(func(event *BlockAcceptedEvent) {
		o.Lock()
		defer o.Unlock()

		delete(o.strongChildCounters, event.Block.ID())
	}))
}

func (o *OrphanageManager) OrphanBlock(blockID BlockID, reason error) {
	o.mesh.Storage.Block(blockID).Consume(func(block *Block) {
		o.Events.BlockOrphaned.Trigger(&BlockOrphanedEvent{
			Block:  block,
			Reason: reason,
		})

		for strongParent := range block.ParentsByType(StrongParentType) {
			o.decreaseStrongChildCounter(strongParent)
		}
	})
}

func (o *OrphanageManager) increaseStrongChildCounter(blockID BlockID) {
	o.Lock()
	defer o.Unlock()

	o.strongChildCounters[blockID]++
}

func (o *OrphanageManager) decreaseStrongChildCounter(blockID BlockID) {
	o.Lock()
	defer o.Unlock()

	o.strongChildCounters[blockID]--

	if o.strongChildCounters[blockID] == 0 {
		delete(o.strongChildCounters, blockID)

		o.mesh.Storage.Block(blockID).Consume(func(block *Block) {
			o.Events.AllChildrenOrphaned.Trigger(block)
		})
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region OrphanageManagerEvents ///////////////////////////////////////////////////////////////////////////////////////

type OrphanageManagerEvents struct {
	BlockOrphaned       *event.Event[*BlockOrphanedEvent]
	AllChildrenOrphaned *event.Event[*Block]
}

type BlockOrphanedEvent struct {
	Block  *Block
	Reason error
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
