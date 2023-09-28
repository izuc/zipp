package consensus

import (
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/blockgadget"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/slotgadget"
)

type Events struct {
	BlockGadget *blockgadget.Events
	SlotGadget  *slotgadget.Events

	event.Group[Events, *Events]
}

// NewEvents contains the constructor of the Events object (it is generated by a generic factory).
var NewEvents = event.CreateGroupConstructor(func() (newEvents *Events) {
	return &Events{
		BlockGadget: blockgadget.NewEvents(),
		SlotGadget:  slotgadget.NewEvents(),
	}
})
