package protocol

import (
	"testing"

	"github.com/izuc/zipp.foundation/app/configuration"
	"github.com/izuc/zipp.foundation/app/logger"
	"github.com/izuc/zipp.foundation/crypto/ed25519"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/core/snapshotcreator"
	"github.com/izuc/zipp/packages/network"
	"github.com/izuc/zipp/packages/protocol/congestioncontrol"
	"github.com/izuc/zipp/packages/protocol/engine"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/mempool/realitiesledger"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/utxoledger"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/vm"
	"github.com/izuc/zipp/packages/storage/utils"

	"github.com/stretchr/testify/require"
)

// region TestFramework ////////////////////////////////////////////////////////////////////////////////////////////////

const genesisTokenAmount = 100

type TestFramework struct {
	Network  *network.MockedNetwork
	Instance *Protocol
	Local    *identity.Identity

	test    *testing.T
	workers *workerpool.Group

	Engine *engine.TestFramework

	optsProtocolOptions          []options.Option[Protocol]
	optsCongestionControlOptions []options.Option[congestioncontrol.CongestionControl]
}

func NewTestFramework(test *testing.T, workers *workerpool.Group, ledgerVM vm.VM, opts ...options.Option[TestFramework]) *TestFramework {
	_ = logger.InitGlobalLogger(configuration.New())

	return options.Apply(&TestFramework{
		Network: network.NewMockedNetwork(),

		test:    test,
		workers: workers,
	}, opts, func(t *TestFramework) {
		tempDir := utils.NewDirectory(test.TempDir())

		identitiesWeights := map[ed25519.PublicKey]uint64{
			ed25519.GenerateKeyPair().PublicKey: 100,
		}

		ledgerProvider := utxoledger.NewProvider(
			utxoledger.WithMemPoolProvider(
				realitiesledger.NewProvider(
					realitiesledger.WithVM(ledgerVM)),
			),
		)

		require.NoError(test, snapshotcreator.CreateSnapshot(
			snapshotcreator.WithDatabaseVersion(DatabaseVersion),
			snapshotcreator.WithLedgerProvider(ledgerProvider),
			snapshotcreator.WithFilePath(tempDir.Path("snapshot.bin")),
			snapshotcreator.WithGenesisTokenAmount(genesisTokenAmount),
			snapshotcreator.WithGenesisSeed(make([]byte, ed25519.SeedSize)),
			snapshotcreator.WithPledgeIDs(identitiesWeights),
			snapshotcreator.WithAttestAll(true),
		))

		t.Instance = New(workers.CreateGroup("Protocol"), t.Network.Join(identity.GenerateIdentity().ID()), append(t.optsProtocolOptions,
			WithSnapshotPath(tempDir.Path("snapshot.bin")),
			WithBaseDirectory(tempDir.Path()),
			WithLedgerProvider(ledgerProvider),
		)...)

		test.Cleanup(func() {
			t.Instance.Shutdown()
		})

		t.Engine = engine.NewTestFramework(t.test, t.workers.CreateGroup("EngineTestFramework"), t.Instance.Engine())
	})
}

// region Options //////////////////////////////////////////////////////////////////////////////////////////////////////

func WithProtocolOptions(opts ...options.Option[Protocol]) options.Option[TestFramework] {
	return func(t *TestFramework) {
		t.optsProtocolOptions = opts
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
