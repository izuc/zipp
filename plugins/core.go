package plugins

import (
	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/plugins/autopeering"
	"github.com/izuc/zipp/plugins/banner"
	"github.com/izuc/zipp/plugins/blockissuer"
	"github.com/izuc/zipp/plugins/cli"
	"github.com/izuc/zipp/plugins/config"
	"github.com/izuc/zipp/plugins/dashboardmetrics"
	"github.com/izuc/zipp/plugins/faucet"
	"github.com/izuc/zipp/plugins/gracefulshutdown"
	"github.com/izuc/zipp/plugins/indexer"
	"github.com/izuc/zipp/plugins/logger"
	"github.com/izuc/zipp/plugins/manainitializer"
	"github.com/izuc/zipp/plugins/manualpeering"
	"github.com/izuc/zipp/plugins/metrics"
	"github.com/izuc/zipp/plugins/p2p"
	"github.com/izuc/zipp/plugins/peer"
	"github.com/izuc/zipp/plugins/portcheck"
	"github.com/izuc/zipp/plugins/profiling"
	"github.com/izuc/zipp/plugins/profilingrecorder"
	"github.com/izuc/zipp/plugins/protocol"
	"github.com/izuc/zipp/plugins/retainer"
	"github.com/izuc/zipp/plugins/spammer"
	"github.com/izuc/zipp/plugins/warpsync"
)

// Core contains the core plugins of a GoShimmer node.
var Core = node.Plugins(
	banner.Plugin,
	config.Plugin,
	logger.Plugin,
	cli.Plugin,
	gracefulshutdown.Plugin,
	peer.Plugin,
	portcheck.Plugin,
	autopeering.Plugin,
	manualpeering.Plugin,
	profiling.Plugin,
	profilingrecorder.Plugin,
	p2p.Plugin,
	protocol.Plugin,
	retainer.Plugin,
	indexer.Plugin,
	warpsync.Plugin,
	faucet.Plugin,
	blockissuer.Plugin,
	dashboardmetrics.Plugin,
	metrics.Plugin,
	spammer.Plugin,
	manainitializer.Plugin,
)
