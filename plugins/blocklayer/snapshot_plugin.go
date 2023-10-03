package blocklayer

import (
	"context"
	"github.com/izuc/zipp/packages/core/epoch"

	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/node"
	"go.uber.org/dig"

	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/notarization"
	"github.com/izuc/zipp/packages/core/snapshot"
	"github.com/izuc/zipp/packages/node/shutdown"

	"github.com/izuc/zipp/packages/core/mesh_old"
)

const (
	// SnapshotPluginName is the name of the snapshot plugin.
	SnapshotPluginName = "Snapshot"
)

type snapshotPluginDependencies struct {
	dig.In

	Mesh          *mesh_old.Mesh
	Manager         *snapshot.Manager
	NotarizationMgr *notarization.Manager
}

type snapshotDependencies struct {
	dig.In

	NotarizationMgr *notarization.Manager
	Storage         kvstore.KVStore
}

var (
	SnapshotPlugin *node.Plugin
	snapshotDeps   = new(snapshotPluginDependencies)
)

func init() {
	SnapshotPlugin = node.NewPlugin(SnapshotPluginName, snapshotDeps, node.Enabled, configureSnapshotPlugin, runSnapshotPlugin)

	SnapshotPlugin.Events.Init.Hook(event.NewClosure(func(event *node.InitEvent) {
		if err := event.Container.Provide(newSnapshotManager); err != nil {
			SnapshotPlugin.Panic(err)
		}
	}))
}

func configureSnapshotPlugin(plugin *node.Plugin) {
	if Parameters.Snapshot.File != "" {
		emptyHeaderConsumer := func(*ledger.SnapshotHeader) {}
		emptyOutputsConsumer := func([]*ledger.OutputWithMetadata) {}
		emptyEpochDiffsConsumer := func(*ledger.EpochDiff) {}
		emptyActivityLogConsumer := func(activity epoch.SnapshotEpochActivity) {}

		err := snapshot.LoadSnapshot(Parameters.Snapshot.File,
			emptyHeaderConsumer,
			snapshotDeps.Manager.LoadSolidEntryPoints,
			emptyOutputsConsumer,
			emptyEpochDiffsConsumer,
			emptyActivityLogConsumer)
		if err != nil {
			plugin.Panic("could not load snapshot file:", err)
		}
	}

	snapshotDeps.Mesh.ConfirmationOracle.Events().BlockAccepted.Attach(event.NewClosure(func(e *mesh_old.BlockAcceptedEvent) {
		e.Block.ForEachParentByType(mesh_old.StrongParentType, func(parent mesh_old.BlockID) bool {
			index := parent.EpochIndex
			if index < e.Block.ID().EpochIndex {
				snapshotDeps.Manager.InsertSolidEntryPoint(parent)
			}
			return true
		})
	}))

	snapshotDeps.Mesh.ConfirmationOracle.Events().BlockOrphaned.Attach(event.NewClosure(func(event *mesh_old.BlockAcceptedEvent) {
		snapshotDeps.Manager.RemoveSolidEntryPoint(event.Block)
	}))

	snapshotDeps.NotarizationMgr.Events.EpochCommittable.Attach(event.NewClosure(func(e *notarization.EpochCommittableEvent) {
		snapshotDeps.Manager.AdvanceSolidEntryPoints(e.EI)
	}))
}

func runSnapshotPlugin(*node.Plugin) {
	if err := daemon.BackgroundWorker("Snapshot", func(ctx context.Context) {
		<-ctx.Done()
	}, shutdown.PriorityNotarization); err != nil {
		SnapshotPlugin.Panicf("Failed to start as daemon: %s", err)
	}
}

func newSnapshotManager(deps snapshotDependencies) *snapshot.Manager {
	return snapshot.NewManager(deps.NotarizationMgr, NotarizationParameters.SnapshotDepth)
}
