package mesh_old

import (
	"sync"
	"testing"
	"time"

	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/types"

	"github.com/cockroachdb/errors"
	"github.com/izuc/zipp.foundation/core/crypto/ed25519"
	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp/packages/core/mesh_old/payload"
	"github.com/izuc/zipp/packages/core/mesh_old/schedulerutils"
)

// region Scheduler_test /////////////////////////////////////////////////////////////////////////////////////////////

func TestScheduler_StartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()
	mesh.Scheduler.Start()

	time.Sleep(10 * time.Millisecond)
	mesh.Scheduler.Shutdown()
}

func TestScheduler_Submit(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()
	mesh.Scheduler.Start()

	blk := newBlock(selfNode.PublicKey())
	mesh.Storage.StoreBlock(blk)
	assert.NoError(t, mesh.Scheduler.Submit(blk.ID()))
	time.Sleep(100 * time.Millisecond)
	// unsubmit to allow the scheduler to shutdown
	assert.NoError(t, mesh.Scheduler.Unsubmit(blk.ID()))
}

func TestScheduler_updateActiveNodeList(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()
	nodes := make(map[string]*identity.Identity)

	mesh.Scheduler.updateActiveNodesList(map[identity.ID]float64{})
	assert.Equal(t, 0, mesh.Scheduler.buffer.NumActiveNodes())

	for _, node := range []string{"A", "B", "C", "D", "E", "F", "G"} {
		nodes[node] = identity.GenerateIdentity()
	}
	mesh.Scheduler.updateActiveNodesList(map[identity.ID]float64{
		nodes["A"].ID(): 30,
		nodes["B"].ID(): 15,
		nodes["C"].ID(): 25,
		nodes["D"].ID(): 20,
		nodes["E"].ID(): 10,
		nodes["G"].ID(): 0,
	})

	assert.Equal(t, 5, mesh.Scheduler.buffer.NumActiveNodes())
	assert.NotContains(t, mesh.Scheduler.buffer.NodeIDs(), nodes["G"].ID())

	mesh.Scheduler.updateActiveNodesList(map[identity.ID]float64{
		nodes["A"].ID(): 30,
		nodes["B"].ID(): 15,
		nodes["C"].ID(): 25,
		nodes["E"].ID(): 0,
		nodes["F"].ID(): 1,
		nodes["G"].ID(): 5,
	})
	assert.Equal(t, 5, mesh.Scheduler.buffer.NumActiveNodes())
	assert.Contains(t, mesh.Scheduler.buffer.NodeIDs(), nodes["A"].ID())
	assert.Contains(t, mesh.Scheduler.buffer.NodeIDs(), nodes["B"].ID())
	assert.Contains(t, mesh.Scheduler.buffer.NodeIDs(), nodes["C"].ID())
	assert.Contains(t, mesh.Scheduler.buffer.NodeIDs(), nodes["F"].ID())
	assert.Contains(t, mesh.Scheduler.buffer.NodeIDs(), nodes["G"].ID())

	mesh.Scheduler.updateActiveNodesList(map[identity.ID]float64{})
	assert.Equal(t, 0, mesh.Scheduler.buffer.NumActiveNodes())
}

func TestScheduler_Discarded(t *testing.T) {
	t.Skip("Skip test. Zero mana nodes are allowed to issue blocks.")
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	noAManaNode := identity.GenerateIdentity()

	blockDiscarded := make(chan BlockID, 1)
	mesh.Scheduler.Events.BlockDiscarded.Hook(event.NewClosure(func(event *BlockDiscardedEvent) {
		blockDiscarded <- event.BlockID
	}))

	mesh.Scheduler.Start()

	// this node has no mana so the block will be discarded
	blk := newBlock(noAManaNode.PublicKey())
	mesh.Storage.StoreBlock(blk)
	err := mesh.Scheduler.Submit(blk.ID())
	assert.Truef(t, errors.Is(err, schedulerutils.ErrInsufficientMana), "unexpected error: %v", err)

	assert.Eventually(t, func() bool {
		select {
		case id := <-blockDiscarded:
			return assert.Equal(t, blk.ID(), id)
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond)
}

func TestScheduler_SetRateBeforeStart(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	mesh.Scheduler.SetRate(time.Hour)
	mesh.Scheduler.Start()
	mesh.Scheduler.SetRate(testRate)
}

func TestScheduler_Schedule(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	blockScheduled := make(chan BlockID, 1)
	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(event *BlockScheduledEvent) {
		blockScheduled <- event.BlockID
	}))

	mesh.Scheduler.Start()

	// create a new block from a different node
	blk := newBlock(peerNode.PublicKey())
	mesh.Storage.StoreBlock(blk)
	assert.NoError(t, mesh.Scheduler.Submit(blk.ID()))
	assert.NoError(t, mesh.Scheduler.Ready(blk.ID()))

	assert.Eventually(t, func() bool {
		select {
		case id := <-blockScheduled:
			return assert.Equal(t, blk.ID(), id)
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond)
}

// MockConfirmationOracleConfirmed mocks ConfirmationOracle marking all blocks as confirmed.
type MockConfirmationOracleConfirmed struct {
	ConfirmationOracle
	events *ConfirmationEvents
}

// IsBlockConfirmed mocks its interface function returning that all blocks are confirmed.
func (m *MockConfirmationOracleConfirmed) IsBlockConfirmed(_ BlockID) bool {
	return true
}

func (m *MockConfirmationOracleConfirmed) Events() *ConfirmationEvents {
	return m.events
}

func TestScheduler_SkipConfirmed(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()
	mesh.ConfirmationOracle = &MockConfirmationOracleConfirmed{
		ConfirmationOracle: mesh.ConfirmationOracle,
		events:             NewConfirmationEvents(),
	}
	blockScheduled := make(chan BlockID, 1)
	blockSkipped := make(chan BlockID, 1)

	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(event *BlockScheduledEvent) {
		blockScheduled <- event.BlockID
	}))
	mesh.Scheduler.Events.BlockSkipped.Hook(event.NewClosure(func(event *BlockSkippedEvent) {
		blockSkipped <- event.BlockID
	}))

	mesh.Scheduler.Setup()

	// create a new block from a different node and mark it as ready and confirmed, but younger than 1 minute
	blkReadyConfirmedNew := newBlock(peerNode.PublicKey())
	mesh.Storage.StoreBlock(blkReadyConfirmedNew)
	assert.NoError(t, mesh.Scheduler.Submit(blkReadyConfirmedNew.ID()))
	assert.NoError(t, mesh.Scheduler.Ready(blkReadyConfirmedNew.ID()))
	assert.Eventually(t, func() bool {
		select {
		case id := <-blockScheduled:
			return assert.Equal(t, blkReadyConfirmedNew.ID(), id)
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond)

	// create a new block from a different node and mark it as unready and confirmed, but younger than 1 minute
	blkUnreadyConfirmedNew := newBlock(peerNode.PublicKey())
	mesh.Storage.StoreBlock(blkUnreadyConfirmedNew)
	assert.NoError(t, mesh.Scheduler.Submit(blkUnreadyConfirmedNew.ID()))
	mesh.ConfirmationOracle.Events().BlockAccepted.Trigger(&BlockAcceptedEvent{blkUnreadyConfirmedNew})
	// make sure that the block was not unsubmitted
	assert.Equal(t, blockIDFromElementID(mesh.Scheduler.buffer.NodeQueue(peerNode.ID()).IDs()[0]), blkUnreadyConfirmedNew.ID())
	assert.NoError(t, mesh.Scheduler.Ready(blkUnreadyConfirmedNew.ID()))
	assert.Eventually(t, func() bool {
		select {
		case id := <-blockScheduled:
			return assert.Equal(t, blkUnreadyConfirmedNew.ID(), id)
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond)

	// create a new block from a different node and mark it as ready and confirmed, but older than 1 minute
	blkReadyConfirmedOld := newBlockWithTimestamp(peerNode.PublicKey(), time.Now().Add(-2*time.Minute))
	mesh.Storage.StoreBlock(blkReadyConfirmedOld)
	assert.NoError(t, mesh.Scheduler.Submit(blkReadyConfirmedOld.ID()))
	assert.NoError(t, mesh.Scheduler.Ready(blkReadyConfirmedOld.ID()))
	assert.Eventually(t, func() bool {
		select {
		case id := <-blockSkipped:
			return assert.Equal(t, blkReadyConfirmedOld.ID(), id)
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond)

	// create a new block from a different node and mark it as unready and confirmed, but older than 1 minute
	blkUnreadyConfirmedOld := newBlockWithTimestamp(peerNode.PublicKey(), time.Now().Add(-2*time.Minute))
	mesh.Storage.StoreBlock(blkUnreadyConfirmedOld)
	assert.NoError(t, mesh.Scheduler.Submit(blkUnreadyConfirmedOld.ID()))
	mesh.ConfirmationOracle.Events().BlockAccepted.Trigger(&BlockAcceptedEvent{blkUnreadyConfirmedOld})

	assert.Eventually(t, func() bool {
		select {
		case id := <-blockSkipped:
			return assert.Equal(t, blkUnreadyConfirmedOld.ID(), id)
		default:
			return false
		}
	}, 1*time.Second, 10*time.Millisecond)
}

func TestScheduler_SetRate(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	var scheduled atomic.Bool
	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(_ *BlockScheduledEvent) {
		scheduled.Store(true)
	}))

	mesh.Scheduler.Start()

	// effectively disable the scheduler by setting a very low rate
	mesh.Scheduler.SetRate(time.Hour)
	// assure that any potential ticks issued before the rate change have been processed
	time.Sleep(100 * time.Millisecond)

	// submit a new block to the scheduler
	blk := newBlock(peerNode.PublicKey())
	mesh.Storage.StoreBlock(blk)
	assert.NoError(t, mesh.Scheduler.Submit(blk.ID()))
	assert.NoError(t, mesh.Scheduler.Ready(blk.ID()))

	// the block should not be scheduled as the rate is too low
	time.Sleep(100 * time.Millisecond)
	assert.False(t, scheduled.Load())

	// after reducing the rate again, the block should eventually be scheduled
	mesh.Scheduler.SetRate(10 * time.Millisecond)
	assert.Eventually(t, scheduled.Load, 1*time.Second, 10*time.Millisecond)
}

func TestScheduler_Time(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	blockScheduled := make(chan BlockID, 1)
	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(event *BlockScheduledEvent) {
		blockScheduled <- event.BlockID
	}))

	mesh.Scheduler.Start()

	future := newBlock(peerNode.PublicKey())
	future.M.IssuingTime = time.Now().Add(time.Second)
	mesh.Storage.StoreBlock(future)
	assert.NoError(t, mesh.Scheduler.Submit(future.ID()))

	now := newBlock(peerNode.PublicKey())
	mesh.Storage.StoreBlock(now)
	assert.NoError(t, mesh.Scheduler.Submit(now.ID()))

	assert.NoError(t, mesh.Scheduler.Ready(future.ID()))
	assert.NoError(t, mesh.Scheduler.Ready(now.ID()))

	done := make(chan struct{})
	scheduledIDs := NewBlockIDs()
	go func() {
		defer close(done)
		timer := time.NewTimer(time.Until(future.IssuingTime()) + 100*time.Millisecond)
		for {
			select {
			case <-timer.C:
				return
			case id := <-blockScheduled:
				mesh.Storage.Block(id).Consume(func(blk *Block) {
					assert.Truef(t, time.Now().After(blk.IssuingTime()), "scheduled too early: %s", time.Until(blk.IssuingTime()))
					scheduledIDs.Add(id)
				})
			}
		}
	}()

	<-done
	assert.Equal(t, NewBlockIDs(now.ID(), future.ID()), scheduledIDs)
}

func TestScheduler_Issue(t *testing.T) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	mesh.Events.Error.Hook(event.NewClosure(func(err error) { assert.Failf(t, "unexpected error", "error event triggered: %v", err) }))

	// setup mesh up till the Scheduler
	mesh.Storage.Setup()
	mesh.Solidifier.Setup()
	mesh.Scheduler.Setup()
	mesh.Solidifier.Events.BlockSolid.Hook(event.NewClosure(func(event *BlockSolidEvent) {
		assert.NoError(t, mesh.Scheduler.SubmitAndReady(event.Block))
	}))
	mesh.Scheduler.Start()

	const numBlocks = 5
	blockScheduled := make(chan BlockID, numBlocks)
	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(event *BlockScheduledEvent) {
		blockScheduled <- event.BlockID
	}))

	ids := NewBlockIDs()
	for i := 0; i < numBlocks; i++ {
		blk := newBlock(selfNode.PublicKey())
		mesh.Storage.StoreBlock(blk)
		ids.Add(blk.ID())
	}

	scheduledIDs := NewBlockIDs()
	assert.Eventually(t, func() bool {
		select {
		case id := <-blockScheduled:
			scheduledIDs.Add(id)
			return len(scheduledIDs) == len(ids)
		default:
			return false
		}
	}, 10*time.Second, 10*time.Millisecond)
	assert.Equal(t, ids, scheduledIDs)
}

func TestSchedulerFlow(t *testing.T) {
	// create Scheduler dependencies
	// create the mesh
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	mesh.Events.Error.Hook(event.NewClosure(func(err error) { assert.Failf(t, "unexpected error", "error event triggered: %v", err) }))

	// setup mesh up till the Scheduler
	mesh.Storage.Setup()
	mesh.Solidifier.Setup()
	mesh.Scheduler.Setup()
	mesh.Solidifier.Events.BlockSolid.Hook(event.NewClosure(func(event *BlockSolidEvent) {
		assert.NoError(t, mesh.Scheduler.SubmitAndReady(event.Block))
	}))
	mesh.Scheduler.Start()

	// testing desired scheduled order: A - B - D - C  (B - A - D - C is equivalent)
	blocks := make(map[string]*Block)
	blocks["A"] = newBlock(selfNode.PublicKey())
	blocks["B"] = newBlock(peerNode.PublicKey())

	// set C to have a timestamp in the future
	blkC := newBlock(selfNode.PublicKey())

	blkC.M.Parents.AddAll(StrongParentType, NewBlockIDs(blocks["A"].ID(), blocks["B"].ID()))

	blkC.M.IssuingTime = time.Now().Add(5 * time.Second)
	blocks["C"] = blkC

	blkD := newBlock(peerNode.PublicKey())
	blkD.M.Parents.AddAll(StrongParentType, NewBlockIDs(blocks["A"].ID(), blocks["B"].ID()))
	blocks["D"] = blkD

	blkE := newBlock(selfNode.PublicKey())
	blkE.M.Parents.AddAll(StrongParentType, NewBlockIDs(blocks["A"].ID(), blocks["B"].ID()))
	blkE.M.IssuingTime = time.Now().Add(3 * time.Second)
	blocks["E"] = blkE

	blockScheduled := make(chan BlockID, len(blocks))
	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(event *BlockScheduledEvent) {
		blockScheduled <- event.BlockID
	}))

	for _, block := range blocks {
		mesh.Storage.StoreBlock(block)
	}

	var scheduledIDs []BlockID
	assert.Eventually(t, func() bool {
		select {
		case id := <-blockScheduled:
			scheduledIDs = append(scheduledIDs, id)
			return len(scheduledIDs) == len(blocks)
		default:
			return false
		}
	}, 10*time.Second, 100*time.Millisecond)
}

func TestSchedulerParallelSubmit(t *testing.T) {
	const (
		totalBlkCount = 200
		meshWidth   = 250
		networkDelay  = 5 * time.Millisecond
	)

	var totalScheduled atomic.Int32

	// create Scheduler dependencies
	// create the mesh
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	mesh.Events.Error.Hook(event.NewClosure(func(err error) { assert.Failf(t, "unexpected error", "error event triggered: %v", err) }))

	// setup mesh up till the Scheduler
	mesh.Storage.Setup()
	mesh.Solidifier.Setup()
	mesh.Scheduler.Setup()
	mesh.Solidifier.Events.BlockSolid.Hook(event.NewClosure(func(event *BlockSolidEvent) {
		assert.NoError(t, mesh.Scheduler.SubmitAndReady(event.Block))
	}))
	mesh.Scheduler.Start()

	// generate the blocks we want to solidify
	blocks := make(map[BlockID]*Block, totalBlkCount)
	for i := 0; i < totalBlkCount/2; i++ {
		blk := newBlock(selfNode.PublicKey())
		blocks[blk.ID()] = blk
	}

	for i := 0; i < totalBlkCount/2; i++ {
		blk := newBlock(peerNode.PublicKey())
		blocks[blk.ID()] = blk
	}

	mesh.Solidifier.Events.BlockSolid.Hook(event.NewClosure(func(event *BlockSolidEvent) {
		t.Logf(event.Block.ID().Base58(), " solid")
	}))

	mesh.Scheduler.Events.BlockScheduled.Hook(event.NewClosure(func(event *BlockScheduledEvent) {
		n := totalScheduled.Add(1)
		t.Logf("scheduled blocks %d/%d", n, totalBlkCount)
	}))

	// issue tips to start solidification
	t.Run("ParallelSubmit", func(t *testing.T) {
		for _, m := range blocks {
			t.Run(m.ID().Base58(), func(t *testing.T) {
				m := m
				t.Parallel()
				t.Logf("issue block: %s", m.ID().Base58())
				mesh.Storage.StoreBlock(m)
			})
		}
	})

	// wait for all blocks to have a formed opinion
	assert.Eventually(t, func() bool { return totalScheduled.Load() == totalBlkCount }, 5*time.Minute, 100*time.Millisecond)
}

func BenchmarkScheduler(b *testing.B) {
	mesh := NewTestMesh(Identity(selfLocalIdentity))
	defer mesh.Shutdown()

	blk := newBlock(selfNode.PublicKey())
	mesh.Storage.StoreBlock(blk)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := mesh.Scheduler.SubmitAndReady(blk); err != nil {
			b.Fatal(err)
		}
		mesh.Scheduler.schedule()
	}
	b.StopTimer()
}

var (
	timeOffset      = 0 * time.Nanosecond
	timeOffsetMutex = sync.Mutex{}
)

func newBlock(issuerPublicKey ed25519.PublicKey) *Block {
	timeOffsetMutex.Lock()
	timeOffset++
	block := NewBlock(
		emptyLikeReferencesFromStrongParents(NewBlockIDs(EmptyBlockID)),
		time.Now().Add(timeOffset),
		issuerPublicKey,
		0,
		payload.NewGenericDataPayload([]byte("")),
		0,
		ed25519.Signature{},
		0,
		epoch.NewECRecord(0),
	)
	timeOffsetMutex.Unlock()
	if err := block.DetermineID(); err != nil {
		panic(err)
	}
	return block
}

func newBlockWithTimestamp(issuerPublicKey ed25519.PublicKey, timestamp time.Time) *Block {
	block := NewBlock(
		ParentBlockIDs{
			StrongParentType: {
				EmptyBlockID: types.Void,
			},
		},
		timestamp,
		issuerPublicKey,
		0,
		payload.NewGenericDataPayload([]byte("")),
		0,
		ed25519.Signature{},
		0,
		epoch.NewECRecord(0),
	)
	if err := block.DetermineID(); err != nil {
		panic(err)
	}
	return block
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
