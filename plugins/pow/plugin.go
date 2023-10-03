package pow

import (
	"github.com/izuc/zipp.foundation/core/logger"
	"github.com/izuc/zipp.foundation/core/node"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/mesh_old"
)

// PluginName is the name of the PoW plugin.
const PluginName = "POW"

var (
	// Plugin is the plugin instance of the PoW plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In

	Mesh           *mesh_old.Mesh
	BlocklayerPlugin *node.Plugin `name:"blocklayer"`
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure)
}

func configure(plugin *node.Plugin) {
	// assure that the logger is available
	log := logger.NewLogger(PluginName)

	if node.IsSkipped(deps.BlocklayerPlugin) {
		log.Infof("%s is disabled; skipping %s\n", deps.BlocklayerPlugin.Name, PluginName)
		return
	}

	// assure that the PoW worker is initialized
	worker := Worker()

	log.Infof("%s started: difficult=%d", PluginName, difficulty)

	deps.Mesh.Parser.AddBytesFilter(mesh_old.NewPowFilter(worker, difficulty))
	deps.Mesh.BlockFactory.SetWorker(mesh_old.WorkerFunc(DoPOW))
	deps.Mesh.BlockFactory.SetTimeout(timeout)
}
