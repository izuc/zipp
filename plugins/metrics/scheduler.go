package metrics

import (
	"time"

	"github.com/izuc/zipp/packages/core/mesh_old/schedulerutils"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/izuc/zipp.foundation/core/syncutils"
)

var (
	// schedulerRate rate at which blocks are scheduled.
	schedulerRate time.Duration

	// readyBlocksCount number of ready blocks in the scheduler buffer.
	readyBlocksCount int

	// totalBlocksCount number of  blocks in the scheduler buffer.
	totalBlocksCount int

	// bufferSize number of bytes waiting to be scheduled.
	bufferSize int

	// schedulerDeficit deficit value
	schedulerDeficit float64
	// maxBufferSize maximum number of blocks can be stored in the buffer.
	maxBufferSize int

	// nodeQueueSizes current size of each node's queue.
	nodeQueueSizes map[identity.ID]int
	// nodeQueueSizes current amount of aMana of each node in the queue.
	nodeAccessMana *schedulerutils.AccessManaCache
	// nodeQueueSizesMutex protect map from concurrent read/write.
	nodeQueueSizesMutex syncutils.RWMutex
)

func measureSchedulerMetrics() {
	nodeQueueSizesMutex.Lock()
	defer nodeQueueSizesMutex.Unlock()
	nodeQueueSizes = make(map[identity.ID]int)
	for k, v := range deps.Mesh.Scheduler.NodeQueueSizes() {
		nodeQueueSizes[k] = v
	}
	if nodeAccessMana == nil {
		nodeAccessMana = deps.Mesh.Scheduler.AccessManaCache()
	}
	bufferSize = deps.Mesh.Scheduler.BufferSize()
	maxBufferSize = deps.Mesh.Options.SchedulerParams.MaxBufferSize
	schedulerDeficit, _ = deps.Mesh.Scheduler.GetDeficit(deps.Local.ID()).Float64()
	schedulerRate = deps.Mesh.Scheduler.Rate()
	readyBlocksCount = deps.Mesh.Scheduler.ReadyBlocksCount()
	totalBlocksCount = deps.Mesh.Scheduler.TotalBlocksCount()
}

// SchedulerNodeQueueSizes current size of each node's queue.
func SchedulerNodeQueueSizes() map[string]int {
	nodeQueueSizesMutex.RLock()
	defer nodeQueueSizesMutex.RUnlock()

	// copy the original map
	clone := make(map[string]int)
	for key, element := range nodeQueueSizes {
		clone[key.String()] = element
	}

	return clone
}

// SchedulerNodeAManaAmount current aMana value for each node in the queue.
func SchedulerNodeAManaAmount() map[string]float64 {
	nodeQueueSizesMutex.RLock()
	defer nodeQueueSizesMutex.RUnlock()

	// copy the original map
	clone := make(map[string]float64)
	for key := range nodeQueueSizes {
		clone[key.String()] = nodeAccessMana.GetCachedMana(key)
	}

	return clone
}

// SchedulerTotalBufferBlocksCount returns if the node is synced based on mesh time.
func SchedulerTotalBufferBlocksCount() int {
	return totalBlocksCount
}

// SchedulerReadyBlocksCount number of ready blocks in the scheduler buffer.
func SchedulerReadyBlocksCount() int {
	return readyBlocksCount
}

// SchedulerMaxBufferSize returns the maximum buffer size.
func SchedulerMaxBufferSize() int {
	return maxBufferSize
}

// SchedulerDeficit local node's deficit value.
func SchedulerDeficit() float64 {
	return schedulerDeficit
}

// SchedulerBufferSize number of bytes waiting to be scheduled.
func SchedulerBufferSize() int {
	return bufferSize
}

// SchedulerRate rate at which blocks are scheduled.
func SchedulerRate() int64 {
	return schedulerRate.Milliseconds()
}
