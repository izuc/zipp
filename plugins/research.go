package plugins

import (
	"github.com/izuc/zipp/packages/node"
	"github.com/izuc/zipp/plugins/activity"
	"github.com/izuc/zipp/plugins/remotelog"
	"github.com/izuc/zipp/plugins/remotemetrics"
)

// Research contains research plugins of a ZIPP node.
var Research = node.Plugins(
	remotelog.Plugin,
	remotemetrics.Plugin,
	activity.Plugin,
)
