package chainmanager

import (
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/core/commitment"
	"github.com/izuc/zipp/packages/protocol/engine/eviction"
)

type Events struct {
	CommitmentMissing         *event.Event1[commitment.ID]
	MissingCommitmentReceived *event.Event1[commitment.ID]
	CommitmentBelowRoot       *event.Event1[commitment.ID]
	ForkDetected              *event.Event1[*Fork]
	EvictionState             *eviction.Events

	event.Group[Events, *Events]
}

var NewEvents = event.CreateGroupConstructor(func() *Events {
	return &Events{
		CommitmentMissing:         event.New1[commitment.ID](),
		MissingCommitmentReceived: event.New1[commitment.ID](),
		CommitmentBelowRoot:       event.New1[commitment.ID](),
		ForkDetected:              event.New1[*Fork](),
	}
})
