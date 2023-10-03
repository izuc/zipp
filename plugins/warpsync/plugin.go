package warpsync

import (
	"context"

	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/node"
	"github.com/pkg/errors"

	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/packages/node/p2p"
	"github.com/izuc/zipp/packages/node/shutdown"
	"github.com/izuc/zipp/packages/node/warpsync"

	"github.com/izuc/zipp/packages/core/notarization"
)

// PluginName is the name of the warpsync plugin.
const PluginName = "Warpsync"

var (
	// Plugin is the plugin instance of the warpsync plugin.
	Plugin *node.Plugin

	deps = new(dependencies)
)

type dependencies struct {
	dig.In

	Mesh          *mesh_old.Mesh
	WarpsyncMgr     *warpsync.Manager
	NotarizationMgr *notarization.Manager
	P2PMgr          *p2p.Manager
}

func init() {
	Plugin = node.NewPlugin(PluginName, deps, node.Disabled, configure, run)

	Plugin.Events.Init.Hook(event.NewClosure(func(event *node.InitEvent) {
		if err := event.Container.Provide(func(t *mesh_old.Mesh, p2pManager *p2p.Manager) *warpsync.Manager {
			// TODO: use a different block loader function
			loadBlockFunc := func(blockID mesh_old.BlockID) (*mesh_old.Block, error) {
				cachedBlock := t.Storage.Block(blockID)
				defer cachedBlock.Release()
				block, exists := cachedBlock.Unwrap()
				if !exists {
					return nil, errors.Errorf("block %s not found", blockID)
				}
				return block, nil
			}
			processBlockFunc := func(blk *mesh_old.Block, peer *peer.Peer) {
				t.Parser.Events.BlockParsed.Trigger(&mesh_old.BlockParsedEvent{
					Block: blk,
					Peer:  peer,
				})
			}
			return warpsync.NewManager(p2pManager, loadBlockFunc, processBlockFunc, Plugin.Logger(), warpsync.WithConcurrency(Parameters.Concurrency), warpsync.WithBlockBatchSize(Parameters.BlockBatchSize))
		}); err != nil {
			Plugin.Panic(err)
		}
	}))
}

func configure(_ *node.Plugin) {
	deps.NotarizationMgr.Events.SyncRange.Attach(event.NewClosure(func(event *notarization.SyncRangeEvent) {
		ctx, cancel := context.WithTimeout(context.Background(), Parameters.SyncRangeTimeOut)
		defer cancel()
		if err := deps.WarpsyncMgr.WarpRange(ctx, event.StartEI, event.EndEI, event.StartEC, event.EndPrevEC); err != nil {
			Plugin.LogWarn("failed to warpsync:", err)
		}
	}))
}

func start(ctx context.Context) {
	defer Plugin.LogInfo("Stopping " + PluginName + " ... done")
	<-ctx.Done()
	Plugin.LogInfo("Stopping " + PluginName + " ...")
}

func run(plugin *node.Plugin) {
	if err := daemon.BackgroundWorker(PluginName, start, shutdown.PriorityWarpsync); err != nil {
		plugin.Logger().Panicf("Failed to start as daemon: %s", err)
	}
}
