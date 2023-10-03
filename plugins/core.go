package plugins

import (
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/plugins/autopeering"
	"github.com/izuc/zipp/plugins/banner"
	"github.com/izuc/zipp/plugins/blocklayer"
	"github.com/izuc/zipp/plugins/bootstrapmanager"
	"github.com/izuc/zipp/plugins/cli"
	"github.com/izuc/zipp/plugins/clock"
	"github.com/izuc/zipp/plugins/config"
	"github.com/izuc/zipp/plugins/database"
	"github.com/izuc/zipp/plugins/faucet"
	"github.com/izuc/zipp/plugins/firewall"
	"github.com/izuc/zipp/plugins/gossip"
	"github.com/izuc/zipp/plugins/gracefulshutdown"
	"github.com/izuc/zipp/plugins/logger"
	"github.com/izuc/zipp/plugins/manaeventlogger"
	"github.com/izuc/zipp/plugins/manainitializer"
	"github.com/izuc/zipp/plugins/manualpeering"
	"github.com/izuc/zipp/plugins/metrics"
	"github.com/izuc/zipp/plugins/p2p"
	"github.com/izuc/zipp/plugins/peer"
	"github.com/izuc/zipp/plugins/portcheck"
	"github.com/izuc/zipp/plugins/pow"
	"github.com/izuc/zipp/plugins/profiling"
	"github.com/izuc/zipp/plugins/spammer"
	"github.com/izuc/zipp/plugins/warpsync"
)

// Core contains the core plugins of a ZIPP node.
var Core = node.Plugins(
	banner.Plugin,
	config.Plugin,
	logger.Plugin,
	cli.Plugin,
	gracefulshutdown.Plugin,
	database.Plugin,
	peer.Plugin,
	portcheck.Plugin,
	autopeering.Plugin,
	manualpeering.Plugin,
	profiling.Plugin,
	pow.Plugin,
	clock.Plugin,
	blocklayer.Plugin,
	p2p.Plugin,
	gossip.Plugin,
	warpsync.Plugin,
	firewall.Plugin,
	blocklayer.ManaPlugin,
	blocklayer.NotarizationPlugin,
	blocklayer.SnapshotPlugin,
	bootstrapmanager.Plugin,
	faucet.Plugin,
	metrics.Plugin,
	spammer.Plugin,
	manaeventlogger.Plugin,
	manainitializer.Plugin,
)
