package ratelimiter

import (
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/runtime/event"
)

type Events struct {
	Hit *event.Event1[*HitEvent]
}

func newEvents() *Events {
	return &Events{
		Hit: event.New1[*HitEvent](),
	}
}

type HitEvent struct {
	Source    identity.ID
	RateLimit *RateLimit
}
