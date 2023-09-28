package blockissuer

import (
	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/autopeering/peer"
	"github.com/izuc/zipp.foundation/runtime/event"
	"github.com/izuc/zipp/packages/app/blockissuer"
	"github.com/izuc/zipp/packages/app/blockissuer/blockfactory"
	"github.com/izuc/zipp/packages/app/blockissuer/ratesetter"
	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/packages/protocol"
	protocolParams "github.com/izuc/zipp/plugins/protocol"
)

// PluginName is the name of the spammer plugin.
const PluginName = "BlockIssuer"

var (
	// Plugin is the plugin instance of the spammer plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In

	BlockIssuer *blockissuer.BlockIssuer
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure)

	Plugin.Events.Init.Hook(func(event *node.InitEvent) {
		if err := event.Container.Provide(createBlockIssuer); err != nil {
			Plugin.Panic(err)
		}
	})
}

func configure(plugin *node.Plugin) {
	deps.BlockIssuer.Events.Error.Hook(func(err error) {
		Plugin.LogErrorf("Error in BlockIssuer: %s", err)
	}, event.WithWorkerPool(plugin.WorkerPool))
}

func createBlockIssuer(local *peer.Local, protocol *protocol.Protocol) *blockissuer.BlockIssuer {
	rateSetterMode := ratesetter.ParseRateSetterMode(Parameters.RateSetter.Mode)
	rateSetter := ratesetter.New(local.ID(), protocol,
		ratesetter.WithMode(rateSetterMode),
		ratesetter.WithInitialRate(Parameters.RateSetter.Initial),
		ratesetter.WithPause(Parameters.RateSetter.Pause),
		ratesetter.WithSchedulerRate(protocolParams.SchedulerParameters.Rate),
	)

	return blockissuer.New(protocol, local.LocalIdentity(),
		blockissuer.WithBlockFactoryOptions(
			blockfactory.WithTipSelectionRetryInterval(Parameters.BlockFactory.TipSelectionRetryInterval),
			blockfactory.WithTipSelectionTimeout(Parameters.BlockFactory.TipSelectionTimeout),
		),
		blockissuer.WithRateSetter(rateSetter),
		blockissuer.WithIgnoreBootstrappedFlag(Parameters.IgnoreBootstrappedFlag),
		blockissuer.WithTimeSinceConfirmationThreshold(protocolParams.Parameters.TimeSinceConfirmationThreshold),
	)
}
