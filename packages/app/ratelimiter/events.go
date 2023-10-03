package ratelimiter

import (
	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/generics/event"
)

type Events struct {
	Hit *event.Event[*HitEvent]
}

func newEvents() (new *Events) {
	return &Events{
		Hit: event.New[*HitEvent](),
	}
}

type HitEvent struct {
	Peer      *peer.Peer
	RateLimit *RateLimit
}
