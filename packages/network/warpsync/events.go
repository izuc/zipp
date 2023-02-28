package warpsync

import (
	"github.com/iotaledger/goshimmer/packages/core/commitment"
	"github.com/iotaledger/goshimmer/packages/core/slot"
	"github.com/iotaledger/goshimmer/packages/protocol/models"
	"github.com/iotaledger/hive.go/core/identity"
	"github.com/iotaledger/hive.go/runtime/event"
)

// Events defines all the events related to the gossip protocol.
type Events struct {
	// Fired when a new block was received via the gossip protocol.
	SlotCommitmentReceived    *event.Event1[*SlotCommitmentReceivedEvent]
	SlotBlocksRequestReceived *event.Event1[*SlotBlocksRequestReceivedEvent]
	SlotBlocksStart           *event.Event1[*SlotBlocksStartEvent]
	SlotBlock                 *event.Event1[*SlotBlockEvent]
	SlotBlocksEnd             *event.Event1[*SlotBlocksEndEvent]

	event.Group[Events, *Events]
}

// NewEvents contains the constructor of the Events object (it is generated by a generic factory).
var NewEvents = event.CreateGroupConstructor(func() (newEvents *Events) {
	return &Events{
		SlotCommitmentReceived: event.New1[*SlotCommitmentReceivedEvent](),
	}
})

// SlotCommitmentReceivedEvent holds data about a slot commitment received event.
type SlotCommitmentReceivedEvent struct {
	ID       identity.ID
	SCRecord *commitment.Commitment
}

// SlotBlocksRequestReceivedEvent holds data about a slot blocks request received event.
type SlotBlocksRequestReceivedEvent struct {
	ID identity.ID
	SI slot.Index
	SC commitment.ID
}

// SlotBlocksStartEvent holds data about a slot blocks start event.
type SlotBlocksStartEvent struct {
	ID identity.ID
	SI slot.Index
}

// SlotBlockEvent holds data about a slot block event.
type SlotBlockEvent struct {
	ID    identity.ID
	SI    slot.Index
	Block *models.Block
}

// SlotBlocksEndEvent holds data about a slot blocks end event.
type SlotBlocksEndEvent struct {
	ID    identity.ID
	SI    slot.Index
	SC    commitment.ID
	Roots *commitment.Roots
}
