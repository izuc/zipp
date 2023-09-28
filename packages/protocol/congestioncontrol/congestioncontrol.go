package congestioncontrol

import (
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/syncutils"
	"github.com/izuc/zipp/packages/protocol/congestioncontrol/icca/scheduler"
	"github.com/izuc/zipp/packages/protocol/engine"
	"github.com/izuc/zipp/packages/protocol/models"
)

type CongestionControl struct {
	Events *Events

	scheduler      *scheduler.Scheduler
	schedulerMutex syncutils.RWMutexFake

	optsSchedulerOptions []options.Option[scheduler.Scheduler]
}

func New(opts ...options.Option[CongestionControl]) *CongestionControl {
	return options.Apply(&CongestionControl{
		Events: NewEvents(),
	}, opts)
}

func (c *CongestionControl) Shutdown() {
	c.schedulerMutex.RLock()
	defer c.schedulerMutex.RUnlock()

	c.scheduler.Shutdown()
}

func (c *CongestionControl) LinkTo(engine *engine.Engine) {
	c.schedulerMutex.Lock()
	defer c.schedulerMutex.Unlock()

	if c.scheduler != nil {
		c.scheduler.Shutdown()
	}

	c.scheduler = scheduler.New(
		engine.EvictionState,
		engine.SlotTimeProvider(),
		engine.Consensus.BlockGadget().IsBlockAccepted,
		engine.ThroughputQuota.BalanceByIDs,
		engine.ThroughputQuota.TotalBalance,
		c.optsSchedulerOptions...,
	)
	c.Events.Scheduler.LinkTo(c.scheduler.Events)

	//wp := engine.Workers.CreatePool("Scheduler", 1)
	engine.HookStopped(lo.Batch(
		engine.Events.Tangle.Booker.BlockTracked.Hook(c.scheduler.AddBlock /*, event.WithWorkerPool(wp)*/).Unhook,
		engine.Events.Tangle.BlockDAG.BlockOrphaned.Hook(c.scheduler.HandleOrphanedBlock /*, event.WithWorkerPool(wp)*/).Unhook,
		engine.Consensus.Events().BlockGadget.BlockAccepted.Hook(c.scheduler.HandleAcceptedBlock /*, event.WithWorkerPool(wp)*/).Unhook,
	))

	c.scheduler.Start()
}

func (c *CongestionControl) Scheduler() *scheduler.Scheduler {
	c.schedulerMutex.RLock()
	defer c.schedulerMutex.RUnlock()

	return c.scheduler
}

func (c *CongestionControl) Block(id models.BlockID) (block *scheduler.Block, exists bool) {
	c.schedulerMutex.RLock()
	defer c.schedulerMutex.RUnlock()

	return c.scheduler.Block(id)
}

func WithSchedulerOptions(opts ...options.Option[scheduler.Scheduler]) options.Option[CongestionControl] {
	return func(c *CongestionControl) {
		c.optsSchedulerOptions = opts
	}
}
