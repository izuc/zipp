package bootstrapmanager

import (
	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/packages/core/bootstrapmanager"
	"github.com/izuc/zipp/packages/core/notarization"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

// region Plugin ///////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	// Plugin is the plugin instance of the blocklayer plugin.
	Plugin     *node.Plugin
	_          = new(dependencies)
	pluginDeps = new(pluginDependencies)
)

type dependencies struct {
	dig.In

	Mesh          *mesh_old.Mesh
	NotarizationMgr *notarization.Manager
}

type pluginDependencies struct {
	dig.In

	Mesh           *mesh_old.Mesh
	NotarizationMgr  *notarization.Manager
	BootstrapManager *bootstrapmanager.Manager
}

func init() {
	Plugin = node.NewPlugin("BootstrapManager", pluginDeps, node.Enabled, configure)

	Plugin.Events.Init.Hook(event.NewClosure(func(event *node.InitEvent) {
		if err := event.Container.Provide(newManager); err != nil {
			Plugin.Panic(err)
		}
	}))
}

func configure(_ *node.Plugin) {
	pluginDeps.BootstrapManager.Setup()
}

func newManager(deps dependencies) *bootstrapmanager.Manager {
	return bootstrapmanager.New(
		deps.Mesh,
		deps.NotarizationMgr,
	)
}
