package plugins

import (
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/plugins/activity"
	analysisclient "github.com/izuc/zipp/plugins/analysis/client"
	analysisdashboard "github.com/izuc/zipp/plugins/analysis/dashboard"
	analysisserver "github.com/izuc/zipp/plugins/analysis/server"
	"github.com/izuc/zipp/plugins/chat"
	"github.com/izuc/zipp/plugins/epochstorage"
	"github.com/izuc/zipp/plugins/networkdelay"
	"github.com/izuc/zipp/plugins/prometheus"
	"github.com/izuc/zipp/plugins/remotelog"
	"github.com/izuc/zipp/plugins/remotemetrics"
)

// Research contains research plugins of a ZIPP node.
var Research = node.Plugins(
	remotelog.Plugin,
	analysisserver.Plugin,
	analysisclient.Plugin,
	analysisdashboard.Plugin,
	prometheus.Plugin,
	remotemetrics.Plugin,
	epochstorage.Plugin,
	networkdelay.App(),
	activity.Plugin,
	chat.Plugin,
)
