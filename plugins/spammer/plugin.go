package spammer

import (
	"context"

	"github.com/labstack/echo/v4"
	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/app/daemon"
	"github.com/izuc/zipp.foundation/logger"
	"github.com/izuc/zipp/packages/app/blockissuer"
	"github.com/izuc/zipp/packages/app/spammer"
	"github.com/izuc/zipp/packages/core/shutdown"
	"github.com/izuc/zipp/packages/node"
)

var blockSpammer *spammer.Spammer

// PluginName is the name of the spammer plugin.
const PluginName = "Spammer"

var (
	// Plugin is the plugin instance of the spammer plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
	log    *logger.Logger
)

type dependencies struct {
	dig.In

	BlockIssuer *blockissuer.BlockIssuer
	Server      *echo.Echo
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Disabled, configure, run)
}

func configure(_ *node.Plugin) {
	log = logger.NewLogger(PluginName)

	blockSpammer = spammer.New(deps.BlockIssuer.IssuePayload, log, deps.BlockIssuer.Estimate)
	deps.Server.GET("spammer", handleRequest)
}

func run(*node.Plugin) {
	if err := daemon.BackgroundWorker("spammer", func(ctx context.Context) {
		<-ctx.Done()

		blockSpammer.Shutdown()
	}, shutdown.PrioritySpammer); err != nil {
		log.Panicf("Failed to start as daemon: %s", err)
	}
}
