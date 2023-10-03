package plugins

import (
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/plugins/webapi"
	"github.com/izuc/zipp/plugins/webapi/autopeering"
	"github.com/izuc/zipp/plugins/webapi/block"
	"github.com/izuc/zipp/plugins/webapi/data"
	"github.com/izuc/zipp/plugins/webapi/epoch"
	"github.com/izuc/zipp/plugins/webapi/faucet"
	"github.com/izuc/zipp/plugins/webapi/faucetrequest"
	"github.com/izuc/zipp/plugins/webapi/healthz"
	"github.com/izuc/zipp/plugins/webapi/info"
	"github.com/izuc/zipp/plugins/webapi/ledgerstate"
	"github.com/izuc/zipp/plugins/webapi/mana"
	"github.com/izuc/zipp/plugins/webapi/ratesetter"
	"github.com/izuc/zipp/plugins/webapi/scheduler"
	"github.com/izuc/zipp/plugins/webapi/snapshot"
	"github.com/izuc/zipp/plugins/webapi/weightprovider"
)

// WebAPI contains the webapi endpoint plugins of a ZIPP node.
var WebAPI = node.Plugins(
	webapi.Plugin,
	data.Plugin,
	faucetrequest.Plugin,
	faucet.Plugin,
	healthz.Plugin,
	block.Plugin,
	autopeering.Plugin,
	info.Plugin,
	epoch.Plugin,
	mana.Plugin,
	ledgerstate.Plugin,
	snapshot.Plugin,
	weightprovider.Plugin,
	ratesetter.Plugin,
	scheduler.Plugin,
)
