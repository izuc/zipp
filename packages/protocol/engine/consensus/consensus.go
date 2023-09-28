package consensus

import (
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/blockgadget"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/conflictresolver"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/slotgadget"
)

type Consensus interface {
	Events() *Events

	BlockGadget() blockgadget.Gadget

	SlotGadget() slotgadget.Gadget

	ConflictResolver() *conflictresolver.ConflictResolver

	module.Interface
}
