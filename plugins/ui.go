package plugins

import (
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/plugins/dagsvisualizer"
	"github.com/izuc/zipp/plugins/dashboard"
)

// UI contains the user interface plugins of a ZIPP node.
var UI = node.Plugins(
	dagsvisualizer.Plugin,
	dashboard.Plugin,
)
