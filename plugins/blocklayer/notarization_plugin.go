package blocklayer

import (
	"context"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/node"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/notarization"
	"github.com/izuc/zipp/packages/core/snapshot"
	"github.com/izuc/zipp/packages/node/shutdown"

	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

const (
	// NotarizationPluginName is the name of the notarization plugin.
	NotarizationPluginName = "Notarization"
)

type notarizationPluginDependencies struct {
	dig.In

	Mesh  *mesh_old.Mesh
	Manager *notarization.Manager
}

type notarizationManagerDependencies struct {
	dig.In

	Mesh  *mesh_old.Mesh
	Storage kvstore.KVStore
}

var (
	NotarizationPlugin *node.Plugin
	notarizationDeps   = new(notarizationPluginDependencies)
)

func init() {
	NotarizationPlugin = node.NewPlugin(NotarizationPluginName, notarizationDeps, node.Enabled, configureNotarizationPlugin, runNotarizationPlugin)

	NotarizationPlugin.Events.Init.Hook(event.NewClosure(func(event *node.InitEvent) {
		if err := event.Container.Provide(newNotarizationManager); err != nil {
			NotarizationPlugin.Panic(err)
		}
	}))
}

func configureNotarizationPlugin(plugin *node.Plugin) {

	if Parameters.Snapshot.File != "" {
		emptySepsConsumer := func(*snapshot.SolidEntryPoints) {}
		emptyActivityConsumer := func(activityLogs epoch.SnapshotEpochActivity) {}

		err := snapshot.LoadSnapshot(Parameters.Snapshot.File,
			notarizationDeps.Manager.LoadECandEIs,
			emptySepsConsumer,
			notarizationDeps.Manager.LoadOutputsWithMetadata,
			notarizationDeps.Manager.LoadEpochDiff,
			emptyActivityConsumer)
		if err != nil {
			plugin.Panic("could not load snapshot file:", err)
		}
	}
	// attach mana plugin event after notarization manager has been initialized
	notarizationDeps.Manager.Events.ManaVectorUpdate.Hook(onManaVectorToUpdateClosure)
}

func runNotarizationPlugin(*node.Plugin) {
	if err := daemon.BackgroundWorker("Notarization", func(ctx context.Context) {
		<-ctx.Done()
		notarizationDeps.Manager.Shutdown()
	}, shutdown.PriorityNotarization); err != nil {
		NotarizationPlugin.Panicf("Failed to start as daemon: %s", err)
	}
}

func newNotarizationManager(deps notarizationManagerDependencies) *notarization.Manager {
	return notarization.NewManager(
		notarization.NewEpochCommitmentFactory(deps.Storage, deps.Mesh, NotarizationParameters.SnapshotDepth),
		deps.Mesh,
		notarization.MinCommittableEpochAge(NotarizationParameters.MinEpochCommittableAge),
		notarization.BootstrapWindow(NotarizationParameters.BootstrapWindow),
		notarization.ManaEpochDelay(ManaParameters.EpochDelay),
		notarization.Log(Plugin.Logger()))
}

// GetLatestEC returns the latest commitment that a new block should commit to.
func GetLatestEC() (ecRecord *epoch.ECRecord, latestConfirmedEpoch epoch.Index, err error) {
	ecRecord, err = notarizationDeps.Manager.GetLatestEC()
	if err != nil {
		return
	}
	latestConfirmedEpoch, err = notarizationDeps.Manager.LatestConfirmedEpochIndex()
	return
}
