package filter

import (
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/protocol/models"
)

type Events struct {
	BlockFiltered *event.Event1[*BlockFilteredEvent]
	BlockAllowed  *event.Event1[*models.Block]

	event.Group[Events, *Events]
}

var NewEvents = event.CreateGroupConstructor(func() *Events {
	return &Events{
		BlockFiltered: event.New1[*BlockFilteredEvent](),
		BlockAllowed:  event.New1[*models.Block](),
	}
})

type BlockFilteredEvent struct {
	Block  *models.Block
	Reason error
}
