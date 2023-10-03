package gossip

import (
	"github.com/izuc/zipp.foundation/core/generics/lo"
	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/packages/node/gossip"
	"github.com/izuc/zipp/packages/node/p2p"
	"github.com/izuc/zipp/packages/node/shutdown"

	"github.com/izuc/zipp/packages/core/mesh_old"
)

// PluginName is the name of the gossip plugin.
const PluginName = "Gossip"

var (
	// Plugin is the plugin instance of the gossip plugin.
	Plugin *node.Plugin

	deps = new(dependencies)
)

type dependencies struct {
	dig.In

	Mesh    *mesh_old.Mesh
	GossipMgr *gossip.Manager
	P2PMgr    *p2p.Manager
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Enabled, configure, run)

	Plugin.Events.Init.Hook(event.NewClosure(func(event *node.InitEvent) {
		if err := event.Container.Provide(createManager); err != nil {
			Plugin.Panic(err)
		}
	}))
}

func configure(_ *node.Plugin) {
	configureLogging()
	configureBlockLayer()
}

func run(plugin *node.Plugin) {
	if err := daemon.BackgroundWorker(PluginName, start, shutdown.PriorityGossip); err != nil {
		plugin.Logger().Panicf("Failed to start as daemon: %s", err)
	}
}

func configureLogging() {
	// log the gossip events
	deps.Mesh.Requester.Events.RequestStarted.Attach(event.NewClosure(func(event *mesh_old.RequestStartedEvent) {
		Plugin.LogDebugf("started to request missing Block with %s", event.BlockID)
	}))
	deps.Mesh.Requester.Events.RequestStopped.Attach(event.NewClosure(func(event *mesh_old.RequestStoppedEvent) {
		Plugin.LogDebugf("stopped to request missing Block with %s", event.BlockID)
	}))
	deps.Mesh.Requester.Events.RequestFailed.Attach(event.NewClosure(func(event *mesh_old.RequestFailedEvent) {
		Plugin.LogDebugf("failed to request missing Block with %s", event.BlockID)
	}))
}

func configureBlockLayer() {
	// configure flow of incoming blocks
	deps.GossipMgr.Events.BlockReceived.Attach(event.NewClosure(func(event *gossip.BlockReceivedEvent) {
		deps.Mesh.ProcessGossipBlock(event.Data, event.Peer)
	}))

	// configure flow of outgoing blocks (gossip upon dispatched blocks)
	deps.Mesh.Scheduler.Events.BlockScheduled.Attach(event.NewClosure(func(event *mesh_old.BlockScheduledEvent) {
		deps.Mesh.Storage.Block(event.BlockID).Consume(func(block *mesh_old.Block) {
			deps.GossipMgr.SendBlock(lo.PanicOnErr(block.Bytes()))
		})
	}))

	// request missing blocks
	deps.Mesh.Requester.Events.RequestIssued.Attach(event.NewClosure(func(event *mesh_old.RequestIssuedEvent) {
		id := event.BlockID
		Plugin.LogDebugf("requesting missing Block with %s", id)

		deps.GossipMgr.RequestBlock(id.Bytes())
	}))
}
