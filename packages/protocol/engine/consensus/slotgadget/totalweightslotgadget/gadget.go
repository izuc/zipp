package totalweightslotgadget

import (
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/syncutils"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/core/votes/slottracker"
	"github.com/izuc/zipp/packages/protocol/engine"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/slotgadget"
	"github.com/izuc/zipp/packages/protocol/engine/tangle"
)

type Gadget struct {
	events  *slotgadget.Events
	workers *workerpool.Group

	tangle              tangle.Tangle
	lastConfirmedSlot   slot.Index
	totalWeightCallback func() int64

	mutex syncutils.RWMutexFake

	optsSlotConfirmationThreshold float64

	module.Module
}

func NewProvider(opts ...options.Option[Gadget]) module.Provider[*engine.Engine, slotgadget.Gadget] {
	return module.Provide(func(e *engine.Engine) slotgadget.Gadget {
		return options.Apply(&Gadget{
			events:                        slotgadget.NewEvents(),
			optsSlotConfirmationThreshold: 0.67,
		}, opts, func(g *Gadget) {
			e.HookConstructed(func() {
				//g.workers = e.Workers.CreateGroup("SlotGadget")
				g.tangle = e.Tangle
				g.totalWeightCallback = e.SybilProtection.Weights().TotalWeightWithoutZeroIdentity

				e.Events.Tangle.Booker.SlotTracker.VotersUpdated.Hook(func(evt *slottracker.VoterUpdatedEvent) {
					g.refreshSlotConfirmation(evt.PrevLatestSlotIndex, evt.NewLatestSlotIndex)
				} /*, event.WithWorkerPool(g.workers.CreatePool("Refresh", 2))*/)

				e.HookInitialized(func() {
					g.lastConfirmedSlot = e.Storage.Permanent.Settings.LatestConfirmedSlot()
					g.TriggerInitialized()
				})
			})
		},
			(*Gadget).TriggerConstructed,
		)
	})
}

func (g *Gadget) Events() *slotgadget.Events {
	return g.events
}

func (g *Gadget) LastConfirmedSlot() slot.Index {
	g.mutex.RLock()
	defer g.mutex.RUnlock()

	return g.lastConfirmedSlot
}

func (g *Gadget) setLastConfirmedSlot(i slot.Index) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.lastConfirmedSlot = i
}

func (g *Gadget) refreshSlotConfirmation(previousLatestSlotIndex slot.Index, newLatestSlotIndex slot.Index) {
	totalWeight := g.totalWeightCallback()

	for i := lo.Max(g.LastConfirmedSlot(), previousLatestSlotIndex) + 1; i <= newLatestSlotIndex; i++ {
		if !IsThresholdReached(totalWeight, g.tangle.Booker().SlotVotersTotalWeight(i), g.optsSlotConfirmationThreshold) {
			break
		}

		// Lock here, so that SlotVotersTotalWeight is not inside the lock. Otherwise, it might cause a deadlock,
		// because one thread owns write-lock on VirtualVoting lock and needs read lock on SlotGadget lock,
		// while this method holds WriteLock on SlotGadget lock and is waiting for ReadLock on VirtualVoting.
		g.setLastConfirmedSlot(i)

		g.events.SlotConfirmed.Trigger(i)
	}
}

var _ slotgadget.Gadget = new(Gadget)

// region Options //////////////////////////////////////////////////////////////////////////////////////////////////////

func WithSlotConfirmationThreshold(acceptanceThreshold float64) options.Option[Gadget] {
	return func(gadget *Gadget) {
		gadget.optsSlotConfirmationThreshold = acceptanceThreshold
	}
}

func IsThresholdReached(weight, otherWeight int64, threshold float64) bool {
	return otherWeight > int64(float64(weight)*threshold)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
