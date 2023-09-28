package enginemanager_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/izuc/zipp.foundation/crypto/ed25519"
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp.foundation/runtime/workerpool"
	"github.com/izuc/zipp/packages/core/database"
	"github.com/izuc/zipp/packages/core/snapshotcreator"
	"github.com/izuc/zipp/packages/protocol"
	"github.com/izuc/zipp/packages/protocol/engine"
	"github.com/izuc/zipp/packages/protocol/engine/clock/blocktime"
	"github.com/izuc/zipp/packages/protocol/engine/consensus/tangleconsensus"
	"github.com/izuc/zipp/packages/protocol/engine/filter/blockfilter"
	"github.com/izuc/zipp/packages/protocol/engine/ledger/utxoledger"
	"github.com/izuc/zipp/packages/protocol/engine/notarization/slotnotarization"
	"github.com/izuc/zipp/packages/protocol/engine/sybilprotection/dpos"
	"github.com/izuc/zipp/packages/protocol/engine/tangle/inmemorytangle"
	"github.com/izuc/zipp/packages/protocol/engine/throughputquota/mana1"
	"github.com/izuc/zipp/packages/protocol/enginemanager"
	"github.com/izuc/zipp/packages/storage/utils"
)

type EngineManagerTestFramework struct {
	EngineManager *enginemanager.EngineManager

	ActiveEngine *engine.Engine
}

func NewEngineManagerTestFramework(t *testing.T, workers *workerpool.Group, identitiesWeights map[ed25519.PublicKey]uint64) *EngineManagerTestFramework {
	tf := &EngineManagerTestFramework{}

	ledgerProvider := utxoledger.NewProvider()

	snapshotPath := utils.NewDirectory(t.TempDir()).Path("snapshot.bin")

	slotDuration := int64(10)
	require.NoError(t, snapshotcreator.CreateSnapshot(
		snapshotcreator.WithDatabaseVersion(protocol.DatabaseVersion),
		snapshotcreator.WithFilePath(snapshotPath),
		snapshotcreator.WithGenesisTokenAmount(1),
		snapshotcreator.WithGenesisSeed(make([]byte, 32)),
		snapshotcreator.WithPeersPublicKeys(lo.Keys(identitiesWeights)),
		snapshotcreator.WithLedgerProvider(ledgerProvider),
		snapshotcreator.WithPeersAmountsPledged(lo.Values(identitiesWeights)),
		snapshotcreator.WithAttestAll(true),
		snapshotcreator.WithGenesisUnixTime(time.Now().Unix()-slotDuration*10),
		snapshotcreator.WithSlotDuration(slotDuration),
	))

	tf.EngineManager = enginemanager.New(workers.CreateGroup("EngineManager"),
		t.TempDir(),
		protocol.DatabaseVersion,
		[]options.Option[database.Manager]{},
		[]options.Option[engine.Engine]{},
		blocktime.NewProvider(),
		ledgerProvider,
		blockfilter.NewProvider(),
		dpos.NewProvider(),
		mana1.NewProvider(),
		slotnotarization.NewProvider(),
		inmemorytangle.NewProvider(),
		tangleconsensus.NewProvider(),
	)

	var err error
	tf.ActiveEngine, err = tf.EngineManager.LoadActiveEngine()
	require.NoError(t, err)

	require.NoError(t, tf.ActiveEngine.Initialize(snapshotPath))

	return tf
}
