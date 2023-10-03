package activity

import (
	"context"
	"math/rand"
	"time"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/izuc/zipp.foundation/core/timeutil"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/packages/core/mesh_old/payload"

	"github.com/izuc/zipp/packages/node/shutdown"
)

var (
	// Plugin is the plugin instance of the activity plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In
	Mesh *mesh_old.Mesh
}

func init() {
	Plugin = node.NewPlugin("Activity", deps, node.Disabled, configure, run)
}

func configure(plugin *node.Plugin) {
	plugin.LogInfof("starting node with activity plugin")
}

// broadcastActivityBlock broadcasts a sync beacon via communication layer.
func broadcastActivityBlock() {
	activityPayload := payload.NewGenericDataPayload([]byte("activity"))

	// sleep some time according to rate setter estimate
	if deps.Mesh.Options.RateSetterParams.Enabled {
		time.Sleep(deps.Mesh.RateSetter.Estimate())
	}

	blk, err := deps.Mesh.IssuePayload(activityPayload, Parameters.ParentsCount)
	if err != nil {
		Plugin.LogWarnf("error issuing activity block: %s", err)
		return
	}

	Plugin.LogDebugf("issued activity block %s", blk.ID())
}

func run(_ *node.Plugin) {
	if err := daemon.BackgroundWorker("Activity-plugin", func(ctx context.Context) {
		// start with initial delay
		rand.NewSource(time.Now().UnixNano())
		initialDelay := time.Duration(rand.Intn(int(Parameters.DelayOffset)))
		time.Sleep(initialDelay)

		if Parameters.BroadcastInterval > 0 {
			timeutil.NewTicker(broadcastActivityBlock, Parameters.BroadcastInterval, ctx)
		}

		// Wait before terminating, so we get correct log blocks from the daemon regarding the shutdown order.
		<-ctx.Done()
	}, shutdown.PriorityActivity); err != nil {
		Plugin.Panicf("Failed to start as daemon: %s", err)
	}
}
