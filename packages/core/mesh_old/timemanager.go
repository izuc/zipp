package mesh_old

import (
	"context"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/izuc/zipp.foundation/core/cerrors"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/serix"
	"github.com/izuc/zipp.foundation/core/stringify"
	"github.com/izuc/zipp.foundation/core/timeutil"

	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp/packages/node/clock"
)

const (
	lastConfirmedKey = "LastAcceptedBlock"
)

// region TimeManager //////////////////////////////////////////////////////////////////////////////////////////////////

// TimeManager is a Mesh component that keeps track of the MeshTime. The MeshTime can be seen as a clock for the
// entire network as it tracks the time of the last confirmed block. Comparing the issuing time of the last confirmed
// block to the node's current wall clock time then yields a reasonable assessment of how much in sync the node is.
type TimeManager struct {
	Events *TimeManagerEvents

	mesh      *Mesh
	startSynced bool

	lastAcceptedMutex sync.RWMutex
	lastAcceptedBlock LastBlock
	lastSyncedMutex   sync.RWMutex
	lastSynced        bool
	bootstrapped      bool

	ctx    context.Context
	cancel context.CancelFunc
}

// NewTimeManager is the constructor for TimeManager.
func NewTimeManager(mesh *Mesh) *TimeManager {
	t := &TimeManager{
		Events:      newTimeManagerEvents(),
		mesh:      mesh,
		startSynced: mesh.Options.StartSynced,
	}
	t.ctx, t.cancel = context.WithCancel(context.Background())

	// initialize with Genesis
	t.lastAcceptedBlock = LastBlock{
		BlockID:   EmptyBlockID,
		BlockTime: mesh.Options.GenesisTime,
	}

	marshaledLastConfirmedBlock, err := mesh.Options.Store.Get(kvstore.Key(lastConfirmedKey))
	if err != nil && !errors.Is(err, kvstore.ErrKeyNotFound) {
		panic(err)
	}
	// load from storage if key was found
	if marshaledLastConfirmedBlock != nil {
		if t.lastAcceptedBlock, _, err = lastBlockFromBytes(marshaledLastConfirmedBlock); err != nil {
			panic(err)
		}
	}
	// initialize the synced status
	t.lastSynced = t.synced()
	t.bootstrapped = t.lastSynced

	return t
}

// Start starts the TimeManager.
func (t *TimeManager) Start() {
	go t.mainLoop()
}

// Setup sets up the behavior of the component by making it attach to the relevant events of other components.
func (t *TimeManager) Setup() {
	t.mesh.ConfirmationOracle.Events().BlockAccepted.Attach(event.NewClosure(func(event *BlockAcceptedEvent) {
		t.updateTime(event.Block)
		t.updateSyncedState()
	}))
	t.Start()
}

// Shutdown shuts down the TimeManager and persists its state.
func (t *TimeManager) Shutdown() {
	t.lastAcceptedMutex.RLock()
	defer t.lastAcceptedMutex.RUnlock()

	if err := t.mesh.Options.Store.Set(kvstore.Key(lastConfirmedKey), t.lastAcceptedBlock.Bytes()); err != nil {
		t.mesh.Events.Error.Trigger(errors.Errorf("failed to persists LastAcceptedBlock (%v): %w", err, cerrors.ErrFatal))
		return
	}

	// cancel the internal context
	t.cancel()
}

// LastAcceptedBlock returns the last confirmed block.
func (t *TimeManager) LastAcceptedBlock() LastBlock {
	t.lastAcceptedMutex.RLock()
	defer t.lastAcceptedMutex.RUnlock()

	return t.lastAcceptedBlock
}

// LastConfirmedBlock returns the last confirmed block.
func (t *TimeManager) LastConfirmedBlock() LastBlock {
	t.lastAcceptedMutex.RLock()
	defer t.lastAcceptedMutex.RUnlock()

	return t.lastAcceptedBlock
}

// ATT returns the Acceptance Mesh Time, i.e., the issuing time of the last accepted block.
func (t *TimeManager) ATT() time.Time {
	t.lastAcceptedMutex.RLock()
	defer t.lastAcceptedMutex.RUnlock()

	return t.lastAcceptedBlock.BlockTime
}

// CTT returns the confirmed mesh time, i.e. the issuing time of the last confirmed block.
// For now, it's just a stub, it actually returns ATT.
func (t *TimeManager) CTT() time.Time {
	return t.ATT()
}

// RATT return relative acceptance mesh time, i.e., ATT + time since last update of ATT.
func (t *TimeManager) RATT() time.Time {
	timeSinceLastUpdate := time.Now().Sub(t.lastAcceptedTime())
	return t.ATT().Add(timeSinceLastUpdate)
}

// RCTT return relative acceptance mesh time, i.e., CTT + time since last update of CTT.
// For now, it's just a stub, it actually returns RATT.
func (t *TimeManager) RCTT() time.Time {
	return t.RATT()
}

// ActivityTime return the time used for defining nodes' activity window.
func (t *TimeManager) ActivityTime() time.Time {
	// Until we have accepted any block, return static ATT. After accepting anything, return RATT so that the node can recognize nodes that are not active.
	if t.lastAcceptedTime().IsZero() {
		return t.ATT()
	}
	return t.RATT()
}

// Bootstrapped returns whether the node has bootstrapped based on the difference between CTT and the current wall time which can
// be configured via SyncTimeWindow.
// When the node becomes bootstrapped and this method returns true, it can't return false after that.
func (t *TimeManager) Bootstrapped() bool {
	t.lastSyncedMutex.RLock()
	defer t.lastSyncedMutex.RUnlock()
	return t.bootstrapped
}

// Synced returns whether the node is in sync based on the difference between CTT and the current wall time which can
// be configured via SyncTimeWindow.
func (t *TimeManager) Synced() bool {
	t.lastSyncedMutex.RLock()
	defer t.lastSyncedMutex.RUnlock()
	return t.lastSynced
}

func (t *TimeManager) synced() bool {
	if t.startSynced && t.CTT().Unix() == epoch.GenesisTime {
		return true
	}

	return clock.Since(t.CTT()) < t.mesh.Options.SyncTimeWindow
}

// checks whether the synced state needs to be updated and if so,
// triggers a corresponding event reflecting the new state.
func (t *TimeManager) updateSyncedState() {
	t.lastSyncedMutex.Lock()
	defer t.lastSyncedMutex.Unlock()
	if newSynced := t.synced(); t.lastSynced != newSynced {
		t.lastSynced = newSynced
		// trigger the event inside the lock to assure that the status is still correct
		t.Events.SyncChanged.Trigger(&SyncChangedEvent{Synced: newSynced})
		if newSynced {
			t.bootstrapped = true
			t.Events.Bootstrapped.Trigger(&BootstrappedEvent{})
		}
	}
}

// updateTime updates the last confirmed block.
func (t *TimeManager) updateTime(block *Block) {
	t.lastAcceptedMutex.Lock()
	defer t.lastAcceptedMutex.Unlock()

	if t.lastAcceptedBlock.BlockTime.After(block.IssuingTime()) {
		return
	}

	t.lastAcceptedBlock = LastBlock{
		BlockID:    block.ID(),
		BlockTime:  block.IssuingTime(),
		UpdateTime: time.Now(),
	}

	t.Events.AcceptanceTimeUpdated.Trigger(&TimeUpdate{
		BlockID:    t.lastAcceptedBlock.BlockID,
		ATT:        t.lastAcceptedBlock.BlockTime,
		UpdateTime: t.lastAcceptedBlock.UpdateTime,
	})

	t.Events.ConfirmedTimeUpdated.Trigger(&TimeUpdate{
		BlockID:    t.lastAcceptedBlock.BlockID,
		ATT:        t.lastAcceptedBlock.BlockTime,
		UpdateTime: t.lastAcceptedBlock.UpdateTime,
	})
}

func (t *TimeManager) lastAcceptedTime() time.Time {
	t.lastAcceptedMutex.RLock()
	defer t.lastAcceptedMutex.RUnlock()
	return t.lastAcceptedBlock.UpdateTime
}

// the main loop runs the updateSyncedState at least every synced time window interval to keep the synced state updated
// even if no updateTime is ever called.
func (t *TimeManager) mainLoop() {
	timeutil.NewTicker(t.updateSyncedState, func() time.Duration {
		if t.mesh.Options.SyncTimeWindow == 0 {
			return DefaultSyncTimeWindow
		}
		return t.mesh.Options.SyncTimeWindow
	}(), t.ctx).WaitForShutdown()
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region LastAcceptedBlock /////////////////////////////////////////////////////////////////////////////////////////

// LastBlock is a wrapper type for the last confirmed block, consisting of BlockID, BlockTime and UpdateTime.
type LastBlock struct {
	BlockID BlockID `serix:"0"`
	// BlockTime field is the time of the last confirmed block.
	BlockTime time.Time `serix:"1"`
	// UpdateTime field is the time when the last confirmed block was updated.
	UpdateTime time.Time
}

// lastBlockFromBytes unmarshals a LastBlock object from a sequence of bytes.
func lastBlockFromBytes(data []byte) (lcm LastBlock, consumedBytes int, err error) {
	consumedBytes, err = serix.DefaultAPI.Decode(context.Background(), data, &lcm, serix.WithValidation())
	if err != nil {
		err = errors.Errorf("failed to parse Background: %w", err)
		return
	}
	return
}

// Bytes returns a marshaled version of the LastBlock.
func (l LastBlock) Bytes() (marshaledLastConfirmedBlock []byte) {
	objBytes, err := serix.DefaultAPI.Encode(context.Background(), l, serix.WithValidation())
	if err != nil {
		// TODO: what do?
		panic(err)
	}
	return objBytes
}

// String returns a human-readable version of the LastBlock.
func (l LastBlock) String() string {
	return stringify.Struct("LastBlock",
		stringify.NewStructField("BlockID", l.BlockID),
		stringify.NewStructField("BlockTime", l.BlockTime),
		stringify.NewStructField("UpdateTime", l.UpdateTime),
	)
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
