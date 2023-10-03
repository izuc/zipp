package clock

import (
	"context"
	"math/rand"
	"time"

	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/izuc/zipp.foundation/core/timeutil"

	"github.com/izuc/zipp/packages/node/clock"

	"github.com/izuc/zipp/packages/node/shutdown"
)

const (
	maxTries     = 3
	syncInterval = 30 * time.Minute
)

// Plugin is the plugin instance of the clock plugin.
var Plugin *node.Plugin

func init() {
	Plugin = node.NewPlugin("Clock", nil, node.Enabled, configure, run)

	Plugin.Events.Init.Hook(event.NewClosure[*node.InitEvent](func(event *node.InitEvent) {
		if err := event.Container.Provide(func() *node.Plugin {
			return Plugin
		}, dig.Name("clock")); err != nil {
			Plugin.Panic(err)
		}
	}))
}

func configure(plugin *node.Plugin) {
	if len(Parameters.NTPPools) == 0 {
		plugin.LogFatalfAndExit("at least 1 NTP pool needs to be provided to synchronize the local clock.")
	}
}

func run(plugin *node.Plugin) {
	if err := daemon.BackgroundWorker(plugin.Name, func(ctx context.Context) {
		// sync clock on startup
		queryNTPPool()

		// sync clock every 30min to counter drift
		timeutil.NewTicker(queryNTPPool, syncInterval, ctx)

		<-ctx.Done()
	}, shutdown.PrioritySynchronization); err != nil {
		plugin.Panicf("Failed to start as daemon: %s", err)
	}
}

// queryNTPPool queries configured ntpPools for maxTries.
func queryNTPPool() {
	Plugin.LogDebug("Synchronizing clock...")
	for t := maxTries; t > 0; t-- {
		index := rand.Int() % len(Parameters.NTPPools)
		err := clock.FetchTimeOffset(Parameters.NTPPools[index])
		if err == nil {
			Plugin.LogDebug("Synchronizing clock... done")
			return
		}
	}

	Plugin.LogWarn("error while trying to sync clock")
}
