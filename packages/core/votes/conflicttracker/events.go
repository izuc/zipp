package conflicttracker

import (
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/core/votes"
)

type Events[ConflictIDType comparable] struct {
	VoterAdded   *event.Event1[*VoterEvent[ConflictIDType]]
	VoterRemoved *event.Event1[*VoterEvent[ConflictIDType]]

	event.Group[Events[ConflictIDType], *Events[ConflictIDType]]
}

func NewEvents[ConflictIDType comparable](optLinkTargets ...*Events[ConflictIDType]) *Events[ConflictIDType] {
	return event.CreateGroupConstructor(func() (self *Events[ConflictIDType]) {
		return &Events[ConflictIDType]{
			VoterAdded:   event.New1[*VoterEvent[ConflictIDType]](),
			VoterRemoved: event.New1[*VoterEvent[ConflictIDType]](),
		}
	})(optLinkTargets...)
}

type VoterEvent[ConflictIDType comparable] struct {
	Voter      identity.ID
	ConflictID ConflictIDType
	Opinion    votes.Opinion
}
