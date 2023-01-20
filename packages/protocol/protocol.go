package protocol

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/core/byteutils"
	"github.com/iotaledger/hive.go/core/crypto/ed25519"
	"github.com/iotaledger/hive.go/core/generics/event"
	"github.com/iotaledger/hive.go/core/generics/lo"
	"github.com/iotaledger/hive.go/core/generics/options"
	"github.com/iotaledger/hive.go/core/generics/orderedmap"
	"github.com/iotaledger/hive.go/core/generics/set"
	"github.com/iotaledger/hive.go/core/identity"
	"github.com/iotaledger/hive.go/core/serix"
	"github.com/iotaledger/hive.go/core/types"
	"github.com/iotaledger/hive.go/core/workerpool"

	"github.com/iotaledger/goshimmer/packages/core/commitment"
	"github.com/iotaledger/goshimmer/packages/core/database"
	"github.com/iotaledger/goshimmer/packages/core/epoch"
	"github.com/iotaledger/goshimmer/packages/network"
	"github.com/iotaledger/goshimmer/packages/protocol/chainmanager"
	"github.com/iotaledger/goshimmer/packages/protocol/congestioncontrol"
	"github.com/iotaledger/goshimmer/packages/protocol/congestioncontrol/icca/scheduler"
	"github.com/iotaledger/goshimmer/packages/protocol/engine"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/consensus/blockgadget"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/notarization"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/sybilprotection"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/sybilprotection/dpos"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/tangle/blockdag"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/throughputquota"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/throughputquota/mana1"
	"github.com/iotaledger/goshimmer/packages/protocol/ledger"
	"github.com/iotaledger/goshimmer/packages/protocol/models"
	"github.com/iotaledger/goshimmer/packages/protocol/tipmanager"
	"github.com/iotaledger/goshimmer/packages/storage"
	"github.com/iotaledger/goshimmer/packages/storage/utils"
)

const (
	mainBaseDir      = "main"
	candidateBaseDir = "candidate"
)

// region Protocol /////////////////////////////////////////////////////////////////////////////////////////////////////

type Protocol struct {
	Events            *Events
	CongestionControl *congestioncontrol.CongestionControl
	TipManager        *tipmanager.TipManager
	chainManager      *chainmanager.Manager

	dispatcher        network.Endpoint
	networkProtocol   *network.Protocol
	directory         *utils.Directory
	activeEngineMutex sync.RWMutex
	engine            *engine.Engine
	candidateEngine   *engine.Engine
	storage           *storage.Storage
	candidateStorage  *storage.Storage

	optsBaseDirectory    string
	optsSnapshotPath     string
	optsPruningThreshold uint64

	// optsSolidificationOptions []options.Option[solidification.Requester]
	optsCongestionControlOptions      []options.Option[congestioncontrol.CongestionControl]
	optsEngineOptions                 []options.Option[engine.Engine]
	optsTipManagerOptions             []options.Option[tipmanager.TipManager]
	optsStorageDatabaseManagerOptions []options.Option[database.Manager]
	optsSybilProtectionProvider       engine.ModuleProvider[sybilprotection.SybilProtection]
	optsThroughputQuotaProvider       engine.ModuleProvider[throughputquota.ThroughputQuota]
}

func New(dispatcher network.Endpoint, opts ...options.Option[Protocol]) (protocol *Protocol) {
	return options.Apply(&Protocol{
		Events: NewEvents(),

		dispatcher:                  dispatcher,
		optsSybilProtectionProvider: dpos.NewProvider(),
		optsThroughputQuotaProvider: mana1.NewProvider(),

		optsBaseDirectory:    "",
		optsPruningThreshold: 6 * 60, // 1 hour given that epoch duration is 10 seconds
	}, opts,
		(*Protocol).initDirectory,
		(*Protocol).initCongestionControl,
		(*Protocol).initMainChainStorage,
		(*Protocol).initMainEngine,
		(*Protocol).initChainManager,
		(*Protocol).initTipManager,
	)
}

// Run runs the protocol.
func (p *Protocol) Run() {
	p.CongestionControl.Run()
	p.linkTo(p.engine)

	if err := p.engine.Initialize(p.optsSnapshotPath); err != nil {
		panic(err)
	}

	p.initNetworkProtocol()
}

// Shutdown shuts down the protocol.
func (p *Protocol) Shutdown() {
	p.CongestionControl.Shutdown()
	p.engine.Shutdown()
	p.storage.Shutdown()

	if p.candidateEngine != nil {
		p.candidateEngine.Shutdown()
	}

	if p.candidateStorage != nil {
		p.candidateStorage.Shutdown()
	}
}

func (p *Protocol) WorkerPools() map[string]*workerpool.UnboundedWorkerPool {
	wps := make(map[string]*workerpool.UnboundedWorkerPool)
	wps["CongestionControl"] = p.CongestionControl.WorkerPool()
	return lo.MergeMaps(wps, p.engine.WorkerPools())
}

func (p *Protocol) initDirectory() {
	p.directory = utils.NewDirectory(p.optsBaseDirectory)
}

func (p *Protocol) initMainChainStorage() {
	p.storage = storage.New(p.directory.Path(mainBaseDir), DatabaseVersion, p.optsStorageDatabaseManagerOptions...)

	p.Events.Engine.Consensus.EpochGadget.EpochConfirmed.Attach(event.NewClosure(func(epochIndex epoch.Index) {
		p.storage.PruneUntilEpoch(epochIndex - epoch.Index(p.optsPruningThreshold))
	}))
}

func (p *Protocol) initCongestionControl() {
	p.CongestionControl = congestioncontrol.New(p.optsCongestionControlOptions...)

	p.Events.CongestionControl = p.CongestionControl.Events
}

func (p *Protocol) initNetworkProtocol() {
	p.networkProtocol = network.NewProtocol(p.dispatcher)

	p.networkProtocol.Events.BlockRequestReceived.Attach(event.NewClosure(func(event *network.BlockRequestReceivedEvent) {
		if block, exists := p.Engine().Block(event.BlockID); exists {
			p.networkProtocol.SendBlock(block, event.Source)
		}
	}))

	p.networkProtocol.Events.BlockReceived.Attach(event.NewClosure(func(event *network.BlockReceivedEvent) {
		if err := p.ProcessBlock(event.Block, event.Source); err != nil {
			fmt.Print(err)
		}
	}))

	p.networkProtocol.Events.EpochCommitmentReceived.Attach(event.NewClosure(func(event *network.EpochCommitmentReceivedEvent) {
		p.chainManager.ProcessCommitmentFromSource(event.Commitment, event.Source)
	}))

	p.networkProtocol.Events.EpochCommitmentRequestReceived.Attach(event.NewClosure(func(event *network.EpochCommitmentRequestReceivedEvent) {
		if requestedCommitment, _ := p.chainManager.Commitment(event.CommitmentID); requestedCommitment != nil && requestedCommitment.Commitment() != nil {
			p.networkProtocol.SendEpochCommitment(requestedCommitment.Commitment(), event.Source)
		}
	}))

	p.Events.CongestionControl.Scheduler.BlockScheduled.Attach(event.NewClosure(func(block *scheduler.Block) {
		p.networkProtocol.SendBlock(block.ModelsBlock)
	}))

	p.Events.Engine.BlockRequester.Tick.Attach(event.NewClosure(func(blockID models.BlockID) {
		p.networkProtocol.RequestBlock(blockID)
	}))

	p.chainManager.CommitmentRequester.Events.Tick.Attach(event.NewClosure(func(commitmentID commitment.ID) {
		p.networkProtocol.RequestCommitment(commitmentID)
	}))

	p.networkProtocol.Events.AttestationsRequestReceived.Attach(event.NewClosure(func(event *network.AttestationsRequestReceivedEvent) {
		p.ProcessAttestationsRequest(event.Index, event.Source)
	}))

	p.networkProtocol.Events.AttestationsReceived.Attach(event.NewClosure(func(event *network.AttestationsReceivedEvent) {
		p.ProcessAttestations(event.Attestations, event.Source)
	}))
}

func (p *Protocol) initMainEngine() {
	p.engine = engine.New(p.storage, p.optsSybilProtectionProvider, p.optsThroughputQuotaProvider, p.optsEngineOptions...)
}

func (p *Protocol) initChainManager() {
	p.chainManager = chainmanager.NewManager(p.Engine().Storage.Settings.LatestCommitment())

	p.Events.Engine.NotarizationManager.EpochCommitted.Attach(event.NewClosure(func(details *notarization.EpochCommittedDetails) {
		p.chainManager.ProcessCommitment(details.Commitment)
	}))

	p.Events.Engine.Consensus.EpochGadget.EpochConfirmed.Attach(event.NewClosure(func(epochIndex epoch.Index) {
		p.chainManager.CommitmentRequester.EvictUntil(epochIndex)
	}))

	p.Events.chainManager.ForkDetected.Attach(event.NewClosure(func(event *chainmanager.ForkDetectedEvent) {
		p.onForkDetected(event.Commitment, event.StartEpoch(), event.EndEpoch(), event.Source)
	}))

	p.Events.chainManager.LinkTo(p.chainManager.Events)
}

func (p *Protocol) onForkDetected(commitment *commitment.Commitment, startIndex epoch.Index, endIndex epoch.Index, source identity.ID) bool {
	claimedWeight := commitment.CumulativeWeight()
	mainChainCommitment, err := p.Engine().Storage.Commitments.Load(commitment.Index())
	if err != nil {
		p.Events.Error.Trigger(errors.Errorf("failed to load commitment for main chain tip at index %d", commitment.Index()))
		return true
	}

	mainChainWeight := mainChainCommitment.CumulativeWeight()

	if claimedWeight <= mainChainWeight {
		// TODO: ban source?
		p.Events.Error.Trigger(errors.Errorf("dot not process fork with %d CW <= than main chain %d CW received from %s", claimedWeight, mainChainWeight, source))
		return true
	}

	p.networkProtocol.RequestAttestations(startIndex, endIndex, source)
	return false
}

func (p *Protocol) switchEngines() {
	p.activeEngineMutex.Lock()

	// Save a reference to the current main engine and storage so that we can shut it down and prune it after switching
	oldEngineStorage := p.storage
	oldEngine := p.engine

	p.engine = p.candidateEngine
	p.storage = p.candidateStorage

	p.candidateEngine = nil
	p.candidateStorage = nil

	p.linkTo(p.engine)
	//TODO: check if we really need to switch the chainManager
	p.chainManager = chainmanager.NewManager(p.engine.Storage.Settings.LatestCommitment())
	p.chainManager.Events.LinkTo(p.chainManager.Events)

	p.activeEngineMutex.Unlock()

	// Shutdown old engine and storage
	oldEngine.Shutdown()
	oldEngineStorage.Shutdown()

	// Cleanup filesystem
	if err := os.RemoveAll(oldEngineStorage.Directory); err != nil {
		p.Events.Error.Trigger(errors.Wrap(err, "error removing storage directory after switching engines"))
	}
}

func (p *Protocol) initTipManager() {
	// TODO: SWITCH ENGINE SIMILAR TO REQUESTER
	p.TipManager = tipmanager.New(p.CongestionControl.Block, p.optsTipManagerOptions...)

	p.Events.CongestionControl.Scheduler.BlockScheduled.Attach(event.NewClosure(func(block *scheduler.Block) {
		p.TipManager.AddTip(block)
	}))

	p.Events.Engine.EvictionState.EpochEvicted.Attach(event.NewClosure(func(index epoch.Index) {
		p.TipManager.EvictTSCCache(index)
	}))

	p.Events.Engine.Consensus.BlockGadget.BlockAccepted.Attach(event.NewClosure(func(block *blockgadget.Block) {
		p.TipManager.RemoveStrongParents(block.ModelsBlock)
	}))

	p.Events.Engine.Tangle.BlockDAG.BlockOrphaned.Hook(event.NewClosure(func(block *blockdag.Block) {
		if schedulerBlock, exists := p.CongestionControl.Block(block.ID()); exists {
			p.TipManager.DeleteTip(schedulerBlock)
		}
	}))

	p.Events.Engine.Tangle.BlockDAG.BlockUnorphaned.Hook(event.NewClosure(func(block *blockdag.Block) {
		if schedulerBlock, exists := p.CongestionControl.Block(block.ID()); exists {
			p.TipManager.AddTip(schedulerBlock)
		}
	}))

	p.Events.Engine.NotarizationManager.EpochCommitted.Attach(event.NewClosure(func(details *notarization.EpochCommittedDetails) {
		p.TipManager.PromoteFutureTips(details.Commitment)
	}))

	p.Events.Engine.EvictionState.EpochEvicted.Attach(event.NewClosure(func(index epoch.Index) {
		p.TipManager.Evict(index)
	}))

	p.Events.TipManager = p.TipManager.Events
}

func (p *Protocol) ProcessBlock(block *models.Block, src identity.ID) error {
	isSolid, chain, _ := p.chainManager.ProcessCommitmentFromSource(block.Commitment(), src)
	if !isSolid {
		return errors.Errorf("Chain is not solid: ", block.Commitment().ID(), "\nLatest commitment: ", p.storage.Settings.LatestCommitment().ID(), "\nBlock ID: ", block.ID())
	}

	if mainChain := p.storage.Settings.ChainID(); chain.ForkingPoint.ID() == mainChain {
		p.Engine().ProcessBlockFromPeer(block, src)
		return nil
	}

	if candidateEngine, candidateStorage := p.CandidateEngine(), p.CandidateStorage(); candidateEngine != nil && candidateStorage != nil {
		if candidateChain := candidateStorage.Settings.ChainID(); chain.ForkingPoint.ID() == candidateChain {
			candidateEngine.ProcessBlockFromPeer(block, src)

			if candidateEngine.IsBootstrapped() && candidateEngine.Storage.Settings.LatestCommitment().CumulativeWeight() > p.Engine().Storage.Settings.LatestCommitment().CumulativeWeight() {
				p.switchEngines()
			}
			return nil
		}
	}
	return errors.Errorf("block was not processed.")
}

func (p *Protocol) ProcessAttestationsRequest(epochIndex epoch.Index, src identity.ID) {
	attestationsForEpoch, err := p.Engine().NotarizationManager.Attestations.Get(epochIndex)
	if err != nil {
		p.Events.Error.Trigger(errors.Wrapf(err, "failed to get attestations for epoch %d upon request", epochIndex))
		return
	}

	attestationsSet := set.NewAdvancedSet[*notarization.Attestation]()
	attestationsForEpoch.Stream(func(_ ed25519.PublicKey, attestation *notarization.Attestation) bool {
		attestationsSet.Add(attestation)
		return true
	})

	p.networkProtocol.SendAttestations(attestationsSet, src)
}

func (p *Protocol) ProcessAttestations(attestations *orderedmap.OrderedMap[epoch.Index, *set.AdvancedSet[*notarization.Attestation]], source identity.ID) {

	if attestations.Size() == 0 {
		p.Events.Error.Trigger(errors.Errorf("received attestations from peer %s are empty", source.String()))
		return
	}

	_, firstEpochAttestations, exists := attestations.Head()
	if !exists {
		p.Events.Error.Trigger(errors.Errorf("received attestations from peer %s are empty", source.String()))
	}

	attestationsForEpoch, _, exists := firstEpochAttestations.Head()
	if !exists {
		p.Events.Error.Trigger(errors.Errorf("received attestations from peer %s are empty", source.String()))
	}

	forkedEvent, exists := p.chainManager.ForkedEventByForkingPoint(attestationsForEpoch.CommitmentID)
	if !exists {
		p.Events.Error.Trigger(errors.Errorf("failed to get forking point for commitment %s", attestationsForEpoch.CommitmentID))
		return
	}

	// TODO: we have to match the CW of the block with the CW of the obtained attestations

	// Obtain mana vector at forking point - 1
	snapshotIndex := forkedEvent.Chain.ForkingPoint.ID().Index() - 1
	wb := sybilprotection.NewWeightsBatch(snapshotIndex)

	var calculatedCumulativeWeight int64
	var innerError error
	p.Engine().NotarizationManager.PerformLocked(func(m *notarization.Manager) {
		latestCommitment := p.Engine().Storage.Settings.LatestCommitment()

		for i := latestCommitment.Index(); i >= snapshotIndex; i-- {
			p.Engine().LedgerState.StateDiffs.StreamSpentOutputs(i, func(output *ledger.OutputWithMetadata) error {
				if iotaBalance, balanceExists := output.IOTABalance(); balanceExists {
					wb.Update(output.ConsensusManaPledgeID(), int64(iotaBalance))
				}
				return nil
			})

			p.Engine().LedgerState.StateDiffs.StreamCreatedOutputs(i, func(output *ledger.OutputWithMetadata) error {
				if iotaBalance, balanceExists := output.IOTABalance(); balanceExists {
					wb.Update(output.ConsensusManaPledgeID(), -int64(iotaBalance))
				}
				return nil
			})
		}

		calculatedCumulativeWeight = lo.PanicOnErr(p.Engine().Storage.Commitments.Load(snapshotIndex)).CumulativeWeight()
		visitedIdentities := make(map[identity.ID]types.Empty)
		attestations.ForEach(func(epochIndex epoch.Index, epochAttestations *set.AdvancedSet[*notarization.Attestation]) bool {
			for it := epochAttestations.Iterator(); it.HasNext(); {
				attestation := it.Next()

				issuingTimeBytes, err := serix.DefaultAPI.Encode(context.Background(), attestation.IssuingTime, serix.WithValidation())
				if err != nil {
					innerError = errors.Wrap(err, "failed to serialize attestations's issuing time")
					return false
				}

				if !attestation.IssuerPublicKey.VerifySignature(byteutils.ConcatBytes(issuingTimeBytes, lo.PanicOnErr(attestation.CommitmentID.Bytes()), attestation.BlockContentHash[:]), attestation.Signature) {
					innerError = errors.Errorf("invalid attestation signature provided by %s", source)
					return false
				}

				issuerID := identity.NewID(attestation.IssuerPublicKey)
				if _, alreadyVisited := visitedIdentities[issuerID]; alreadyVisited {
					innerError = errors.Errorf("invalid attestation from source %s, issuerID %s contains multiple attestations", source, issuerID)
					//TODO: ban source
					return false
				}

				if weight, weightExists := p.Engine().SybilProtection.Weights().Get(issuerID); weightExists {
					calculatedCumulativeWeight += weight.Value
				}
				calculatedCumulativeWeight += wb.Get(issuerID)

				visitedIdentities[issuerID] = types.Void
			}
			return true
		})
	})
	if innerError != nil {
		p.Events.Error.Trigger(innerError)
		return
	}

	// Add the attestations to the mana vector of the forking point
	// TODO: I think we need to fetch more attestations here, because this is cumulative and we won't reach the same value as in the commitment

	// Compare the CW with our main chain
	if calculatedCumulativeWeight <= p.Engine().Storage.Settings.LatestCommitment().CumulativeWeight() {
		p.Events.Error.Trigger(errors.Errorf("forking point does not accumulate enough weight %s CW <= main chain %s CW", calculatedCumulativeWeight, p.Engine().Storage.Settings.LatestCommitment().CumulativeWeight()))
		return
	}

	// Dump a snapshot at the forking point
	snapshotPath := filepath.Join(os.TempDir(), fmt.Sprintf("snapshot_%d.bin", snapshotIndex))
	if err := p.Engine().WriteSnapshot(snapshotPath, snapshotIndex); err != nil {
		p.Events.Error.Trigger(errors.Wrapf(err, "error exporting snapshot for index %s", snapshotIndex))
		return
	}

	// Initialize a new candidate engine
	//TODO: use unique directories for each new engine
	candidateStorage := storage.New(p.directory.Path(candidateBaseDir), DatabaseVersion, p.optsStorageDatabaseManagerOptions...)
	candidateEngine := engine.New(candidateStorage, p.optsSybilProtectionProvider, p.optsThroughputQuotaProvider, p.optsEngineOptions...)

	if err := candidateEngine.Initialize(snapshotPath); err != nil {
		p.Events.Error.Trigger(errors.Wrap(err, "failed to initialize candidate engine with snapshot"))
		candidateEngine.Shutdown()
		candidateStorage.Shutdown()
		return
	}

	// Set the engine as the new candidate
	p.activeEngineMutex.Lock()
	p.candidateStorage = candidateStorage
	p.candidateEngine = candidateEngine
	p.activeEngineMutex.Unlock()
}

func (p *Protocol) Engine() (instance *engine.Engine) {
	p.activeEngineMutex.RLock()
	defer p.activeEngineMutex.RUnlock()

	return p.engine
}

func (p *Protocol) ChainManager() (instance *chainmanager.Manager) {
	return p.chainManager
}

func (p *Protocol) CandidateEngine() (instance *engine.Engine) {
	p.activeEngineMutex.RLock()
	defer p.activeEngineMutex.RUnlock()

	return p.candidateEngine
}

// MainStorage returns the underlying storage of the main chain.
func (p *Protocol) MainStorage() (mainStorage *storage.Storage) {
	p.activeEngineMutex.RLock()
	defer p.activeEngineMutex.RUnlock()

	return p.storage
}

func (p *Protocol) CandidateStorage() (chainstorage *storage.Storage) {
	p.activeEngineMutex.RLock()
	defer p.activeEngineMutex.RUnlock()

	return p.candidateStorage
}

func (p *Protocol) linkTo(engine *engine.Engine) {
	p.Events.Engine.LinkTo(engine.Events)
	p.TipManager.LinkTo(engine)
	p.CongestionControl.LinkTo(engine)
}

func (p *Protocol) Network() *network.Protocol {
	return p.networkProtocol
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Options //////////////////////////////////////////////////////////////////////////////////////////////////////

func WithBaseDirectory(baseDirectory string) options.Option[Protocol] {
	return func(n *Protocol) {
		n.optsBaseDirectory = baseDirectory
	}
}

func WithPruningThreshold(pruningThreshold uint64) options.Option[Protocol] {
	return func(n *Protocol) {
		n.optsPruningThreshold = pruningThreshold
	}
}

func WithSnapshotPath(snapshot string) options.Option[Protocol] {
	return func(n *Protocol) {
		n.optsSnapshotPath = snapshot
	}
}

func WithSybilProtectionProvider(sybilProtectionProvider engine.ModuleProvider[sybilprotection.SybilProtection]) options.Option[Protocol] {
	return func(n *Protocol) {
		n.optsSybilProtectionProvider = sybilProtectionProvider
	}
}

func WithThroughputQuotaProvider(sybilProtectionProvider engine.ModuleProvider[throughputquota.ThroughputQuota]) options.Option[Protocol] {
	return func(n *Protocol) {
		n.optsThroughputQuotaProvider = sybilProtectionProvider
	}
}

func WithCongestionControlOptions(opts ...options.Option[congestioncontrol.CongestionControl]) options.Option[Protocol] {
	return func(e *Protocol) {
		e.optsCongestionControlOptions = opts
	}
}

func WithTipManagerOptions(opts ...options.Option[tipmanager.TipManager]) options.Option[Protocol] {
	return func(p *Protocol) {
		p.optsTipManagerOptions = opts
	}
}

// func WithSolidificationOptions(opts ...options.Option[solidification.Requester]) options.Option[Protocol] {
// 	return func(n *Protocol) {
// 		n.optsSolidificationOptions = opts
// 	}
// }

func WithEngineOptions(opts ...options.Option[engine.Engine]) options.Option[Protocol] {
	return func(n *Protocol) {
		n.optsEngineOptions = opts
	}
}

func WithStorageDatabaseManagerOptions(opts ...options.Option[database.Manager]) options.Option[Protocol] {
	return func(p *Protocol) {
		p.optsStorageDatabaseManagerOptions = opts
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
