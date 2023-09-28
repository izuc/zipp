package dashboard

import (
	"context"

	"github.com/izuc/zipp.foundation/app/daemon"
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/core/shutdown"
	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/packages/protocol/engine/notarization"
)

type SlotInfo struct {
	Index slot.Index `json:"index"`
	ID    string     `json:"id"`
}

func runSlotsLiveFeed(plugin *node.Plugin) {
	if err := daemon.BackgroundWorker("Dashboard[SlotsLiveFeed]", func(ctx context.Context) {
		hook := deps.Protocol.Events.Engine.Notarization.SlotCommitted.Hook(onSlotCommitted, event.WithWorkerPool(plugin.WorkerPool))

		<-ctx.Done()

		log.Info("Stopping Dashboard[SlotsLiveFeed] ...")
		hook.Unhook()
		log.Info("Stopping Dashboard[SlotsLiveFeed] ... done")
	}, shutdown.PriorityDashboard); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}

func onSlotCommitted(e *notarization.SlotCommittedDetails) {
	broadcastWsBlock(&wsblk{MsgTypeSlotInfo, &SlotInfo{Index: e.Commitment.Index(), ID: e.Commitment.ID().Base58()}})
}
