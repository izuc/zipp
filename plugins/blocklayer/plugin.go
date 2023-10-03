package blocklayer

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/izuc/zipp.foundation/core/generics/lo"
	"go.uber.org/dig"

	"github.com/izuc/zipp.foundation/core/autopeering/discover"
	"github.com/izuc/zipp.foundation/core/autopeering/peer"
	"github.com/izuc/zipp.foundation/core/crypto/ed25519"
	"github.com/izuc/zipp.foundation/core/daemon"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/node"

	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp/packages/core/ledger"
	"github.com/izuc/zipp/packages/core/ledger/utxo"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm/indexer"
	"github.com/izuc/zipp/packages/core/mana"
	"github.com/izuc/zipp/packages/core/mesh_old"

	"github.com/izuc/zipp/packages/core/snapshot"

	"github.com/izuc/zipp/packages/core/consensus/acceptance"
	"github.com/izuc/zipp/packages/core/consensus/omv"
	"github.com/izuc/zipp/packages/core/notarization"
	"github.com/izuc/zipp/packages/node/shutdown"
	"github.com/izuc/zipp/plugins/database"
	"github.com/izuc/zipp/plugins/remotelog"
)

var (
	// ErrBlockWasNotBookedInTime is returned if a block did not get booked within the defined await time.
	ErrBlockWasNotBookedInTime = errors.New("block could not be booked in time")

	// ErrBlockWasNotIssuedInTime is returned if a block did not get issued within the defined await time.
	ErrBlockWasNotIssuedInTime = errors.New("block could not be issued in time")

	snapshotLoadedKey = kvstore.Key("snapshot_loaded")
)

// region Plugin ///////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	// Plugin is the plugin instance of the blocklayer plugin.
	Plugin *node.Plugin
	deps   = new(dependencies)
)

type dependencies struct {
	dig.In

	Mesh           *mesh_old.Mesh
	Indexer          *indexer.Indexer
	Local            *peer.Local
	Discover         *discover.Protocol `optional:"true"`
	Storage          kvstore.KVStore
	RemoteLoggerConn *remotelog.RemoteLoggerConn `optional:"true"`
	NotarizationMgr  *notarization.Manager
}

type meshdeps struct {
	dig.In

	Storage kvstore.KVStore
	Local   *peer.Local
}

type indexerdeps struct {
	dig.In

	Mesh *mesh_old.Mesh
}

func init() {
	Plugin = node.NewPlugin("BlockLayer", deps, node.Enabled, configure, run)

	Plugin.Events.Init.Hook(event.NewClosure(func(event *node.InitEvent) {
		if err := event.Container.Provide(newMesh); err != nil {
			Plugin.Panic(err)
		}

		if err := event.Container.Provide(AcceptanceGadget); err != nil {
			Plugin.Panic(err)
		}

		if err := event.Container.Provide(newIndexer); err != nil {
			Plugin.Panic(err)
		}

		if err := event.Container.Provide(func() *node.Plugin {
			return Plugin
		}, dig.Name("blocklayer")); err != nil {
			Plugin.Panic(err)
		}
	}))
}

func configure(plugin *node.Plugin) {
	deps.Mesh.Events.Error.Attach(event.NewClosure(func(err error) {
		plugin.LogError(err)
	}))

	// Blocks created by the node need to pass through the normal flow.
	deps.Mesh.RateSetter.Events.BlockIssued.Attach(event.NewClosure(func(event *mesh_old.BlockConstructedEvent) {
		deps.Mesh.ProcessGossipBlock(lo.PanicOnErr(event.Block.Bytes()), deps.Local.Peer)
	}))

	deps.Mesh.Booker.Events.BlockBooked.Attach(event.NewClosure(func(event *mesh_old.BlockBookedEvent) {
		deps.Mesh.Storage.Block(event.BlockID).Consume(func(block *mesh_old.Block) {
			ei := epoch.IndexFromTime(block.IssuingTime())
			deps.Mesh.WeightProvider.Update(ei, identity.NewID(block.IssuerPublicKey()))
		})
	}))

	deps.Mesh.Parser.Events.BlockRejected.Attach(event.NewClosure(func(event *mesh_old.BlockRejectedEvent) {
		plugin.LogInfof("block with %s rejected in Parser: %v", event.Block.ID().Base58(), event.Error)
	}))

	deps.Mesh.Parser.Events.BytesRejected.Attach(event.NewClosure(func(event *mesh_old.BytesRejectedEvent) {
		if errors.Is(event.Error, mesh_old.ErrReceivedDuplicateBytes) {
			return
		}

		plugin.LogWarnf("bytes rejected from peer %s: %v", event.Peer.ID(), event.Error)
	}))

	deps.Mesh.Scheduler.Events.BlockDiscarded.Attach(event.NewClosure(func(event *mesh_old.BlockDiscardedEvent) {
		plugin.LogInfof("block rejected in Scheduler: %s", event.BlockID.Base58())
	}))

	deps.Mesh.Scheduler.Events.NodeBlacklisted.Attach(event.NewClosure(func(event *mesh_old.NodeBlacklistedEvent) {
		plugin.LogInfof("node %s is blacklisted in Scheduler", event.NodeID.String())
	}))

	deps.Mesh.TimeManager.Events.SyncChanged.Attach(event.NewClosure(func(event *mesh_old.SyncChangedEvent) {
		plugin.LogInfo("Sync changed: ", event.Synced)
	}))

	// read snapshot file
	if loaded, _ := deps.Storage.Has(snapshotLoadedKey); !loaded && Parameters.Snapshot.File != "" {
		plugin.LogInfof("reading snapshot from %s ...", Parameters.Snapshot.File)

		utxoStatesConsumer := func(outputsWithMetadatas []*ledger.OutputWithMetadata) {
			deps.Mesh.Ledger.LoadOutputWithMetadatas(outputsWithMetadatas)
			for _, outputWithMetadata := range outputsWithMetadatas {
				deps.Indexer.IndexOutput(outputWithMetadata.Output().(devnetvm.Output))
			}
		}

		epochDiffsConsumer := func(epochDiff *ledger.EpochDiff) {
			err := deps.Mesh.Ledger.LoadEpochDiff(epochDiff)
			if err != nil {
				panic(err)
			}
			for _, outputWithMetadata := range epochDiff.Created() {
				deps.Indexer.IndexOutput(outputWithMetadata.Output().(devnetvm.Output))
			}
		}

		emptyHeaderConsumer := func(*ledger.SnapshotHeader) {}
		emptySepsConsumer := func(*snapshot.SolidEntryPoints) {}
		emptyActivityConsumer := func(activity epoch.SnapshotEpochActivity) {}
		err := snapshot.LoadSnapshot(Parameters.Snapshot.File, emptyHeaderConsumer, emptySepsConsumer, utxoStatesConsumer, epochDiffsConsumer, emptyActivityConsumer)
		if err != nil {
			plugin.Panic("could not load snapshot file:", err)
		}

		plugin.LogInfof("reading snapshot from %s ... done", Parameters.Snapshot.File)

		// Set flag that we read the snapshot already, so we don't have to do it again after a restart.
		err = deps.Storage.Set(snapshotLoadedKey, kvstore.Value{})
		if err != nil {
			plugin.LogErrorf("could not store snapshot_loaded flag: %v")
		}
	}

	configureFinality()
}

func run(*node.Plugin) {
	if err := daemon.BackgroundWorker("Mesh", func(ctx context.Context) {
		<-ctx.Done()
		deps.Mesh.Shutdown()
	}, shutdown.PriorityMesh); err != nil {
		Plugin.Panicf("Failed to start as daemon: %s", err)
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Mesh ///////////////////////////////////////////////////////////////////////////////////////////////////////

var meshInstance *mesh_old.Mesh

// newMesh gets the mesh instance.
func newMesh(meshDeps meshdeps) *mesh_old.Mesh {
	// TODO: this should use the time from the snapshot instead of epoch.GenesisTime
	genesisTime := time.Unix(epoch.GenesisTime, 0)
	if Parameters.GenesisTime > 0 {
		genesisTime = time.Unix(Parameters.GenesisTime, 0)
	}

	meshInstance = mesh_old.New(
		mesh_old.Store(meshDeps.Storage),
		mesh_old.Identity(meshDeps.Local.LocalIdentity()),
		mesh_old.Width(Parameters.MeshWidth),
		mesh_old.TimeSinceConfirmationThreshold(Parameters.TimeSinceConfirmationThreshold),
		mesh_old.GenesisNode(Parameters.Snapshot.GenesisNode),
		mesh_old.SchedulerConfig(mesh_old.SchedulerParams{
			MaxBufferSize:                   SchedulerParameters.MaxBufferSize,
			TotalSupply:                     2779530283277761,
			ConfirmedBlockScheduleThreshold: parseDuration(SchedulerParameters.ConfirmedBlockThreshold),
			Rate:                            parseDuration(SchedulerParameters.Rate),
			AccessManaMapRetrieverFunc:      accessManaMapRetriever,
			TotalAccessManaRetrieveFunc:     totalAccessManaRetriever,
		}),
		mesh_old.RateSetterConfig(mesh_old.RateSetterParams{
			Initial:          RateSetterParameters.Initial,
			RateSettingPause: RateSetterParameters.RateSettingPause,
			Enabled:          RateSetterParameters.Enable,
		}),
		mesh_old.GenesisTime(genesisTime),
		mesh_old.SyncTimeWindow(Parameters.MeshTimeWindow),
		mesh_old.StartSynced(Parameters.StartSynced),
		mesh_old.CacheTimeProvider(database.CacheTimeProvider()),
		mesh_old.CommitmentFunc(GetLatestEC),
	)

	meshInstance.Scheduler = mesh_old.NewScheduler(meshInstance)
	meshInstance.WeightProvider = mesh_old.NewCManaWeightProvider(GetCMana, meshInstance.TimeManager.ActivityTime, GetConfirmedEI, meshDeps.Storage)
	meshInstance.OMVConsensusManager = mesh_old.NewOMVConsensusManager(omv.NewOnMeshVoting(meshInstance.Ledger.ConflictDAG, meshInstance.ApprovalWeightManager.WeightOfConflict))

	acceptanceGadget = acceptance.NewSimpleFinalityGadget(meshInstance)
	meshInstance.ConfirmationOracle = acceptanceGadget

	meshInstance.Setup()
	return meshInstance
}

func newIndexer(indexerDeps indexerdeps) *indexer.Indexer {
	return indexer.New(indexerDeps.Mesh.Ledger, indexer.WithStore(indexerDeps.Mesh.Options.Store), indexer.WithCacheTimeProvider(database.CacheTimeProvider()))
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Scheduler ///////////////////////////////////////////////////////////////////////////////////////////

func parseDuration(durationString string) time.Duration {
	duration, err := time.ParseDuration(durationString)
	// if parseDuration failed, scheduler will take default value (5ms)
	if err != nil {
		return 0
	}
	return duration
}

func accessManaMapRetriever() map[identity.ID]float64 {
	nodeMap, _, err := GetManaMap(mana.AccessMana)
	if err != nil {
		return mana.NodeMap{}
	}
	return nodeMap
}

func totalAccessManaRetriever() float64 {
	totalMana, _, err := GetTotalMana(mana.AccessMana)
	if err != nil {
		return 0
	}
	return totalMana
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// AwaitBlockToBeBooked awaits maxAwait for the given block to get booked.
func AwaitBlockToBeBooked(f func() (*mesh_old.Block, error), txID utxo.TransactionID, maxAwait time.Duration) (*mesh_old.Block, error) {
	// first subscribe to the transaction booked event
	booked := make(chan struct{}, 1)
	// exit is used to let the caller exit if for whatever
	// reason the same transaction gets booked multiple times
	exit := make(chan struct{})
	defer close(exit)

	closure := event.NewClosure(func(event *mesh_old.BlockBookedEvent) {
		match := false
		deps.Mesh.Storage.Block(event.BlockID).Consume(func(block *mesh_old.Block) {
			if block.Payload().Type() == devnetvm.TransactionType {
				tx := block.Payload().(*devnetvm.Transaction)
				if tx.ID() == txID {
					match = true
					return
				}
			}
		})
		if !match {
			return
		}
		select {
		case booked <- struct{}{}:
		case <-exit:
		}
	})
	deps.Mesh.Booker.Events.BlockBooked.Attach(closure)
	defer deps.Mesh.Booker.Events.BlockBooked.Detach(closure)

	// then issue the block with the tx
	blk, err := f()

	if err != nil || blk == nil {
		return nil, errors.Errorf("Failed to issue transaction %s: %w", txID.String(), err)
	}

	select {
	case <-time.After(maxAwait):
		return nil, ErrBlockWasNotBookedInTime
	case <-booked:
		return blk, nil
	}
}

// AwaitBlockToBeIssued awaits maxAwait for the given block to get issued.
func AwaitBlockToBeIssued(f func() (*mesh_old.Block, error), issuer ed25519.PublicKey, maxAwait time.Duration) (*mesh_old.Block, error) {
	issued := make(chan *mesh_old.Block, 1)
	exit := make(chan struct{})
	defer close(exit)

	closure := event.NewClosure(func(event *mesh_old.BlockScheduledEvent) {
		deps.Mesh.Storage.Block(event.BlockID).Consume(func(block *mesh_old.Block) {
			if block.IssuerPublicKey() != issuer {
				return
			}
			select {
			case issued <- block:
			case <-exit:
			}
		})
	})
	deps.Mesh.Scheduler.Events.BlockScheduled.Attach(closure)
	defer deps.Mesh.Scheduler.Events.BlockScheduled.Detach(closure)

	// channel to receive the result of issuance
	issueResult := make(chan struct {
		blk *mesh_old.Block
		err error
	}, 1)

	go func() {
		blk, err := f()
		issueResult <- struct {
			blk *mesh_old.Block
			err error
		}{blk: blk, err: err}
	}()

	// wait on issuance
	result := <-issueResult

	if result.err != nil || result.blk == nil {
		return nil, errors.Errorf("Failed to issue data: %w", result.err)
	}

	ticker := time.NewTicker(maxAwait)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			return nil, ErrBlockWasNotIssuedInTime
		case blk := <-issued:
			if result.blk.ID() == blk.ID() {
				return blk, nil
			}
		}
	}
}
