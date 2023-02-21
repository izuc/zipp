package blockgadget

import (
	"github.com/iotaledger/goshimmer/packages/core/memstorage"
	"github.com/iotaledger/goshimmer/packages/protocol/models"
	"github.com/iotaledger/hive.go/runtime/event"
)

type Events struct {
	BlockAccepted  *event.Event1[*Block]
	BlockConfirmed *event.Event1[*Block]
	EpochClosed    *event.Event1[*memstorage.Storage[models.BlockID, *Block]]
	Error          *event.Event1[error]

	event.Group[Events, *Events]
}

// NewEvents contains the constructor of the Events object (it is generated by a generic factory).
var NewEvents = event.CreateGroupConstructor(func() (newEvents *Events) {
	return &Events{
		BlockAccepted:  event.New1[*Block](),
		BlockConfirmed: event.New1[*Block](),
		EpochClosed:    event.New1[*memstorage.Storage[models.BlockID, *Block]](),
		Error:          event.New1[error](),
	}
})
