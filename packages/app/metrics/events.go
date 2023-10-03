package metrics

import "github.com/izuc/zipp.foundation/core/generics/event"

// CollectionEvents defines the events fot the metrics package.
type CollectionEvents struct {
	// AnalysisOutboundBytes defines the local analysis outbound network traffic in bytes.
	AnalysisOutboundBytes *event.Event[*AnalysisOutboundBytesEvent]
	// CPUUsage defines the local CPU usage.
	CPUUsage *event.Event[*CPUUsageEvent]
	// MemUsage defines the local ZIPP memory usage.
	MemUsage *event.Event[*MemUsageEvent]
	// MeshTimeSynced defines the local sync status event based on mesh time.
	MeshTimeSynced *event.Event[*MeshTimeSyncedEvent]
	// ValueTips defines the local value tips count event.
	ValueTips *event.Event[*ValueTipsEvent]
	// BlockTips defines the local block tips count event.
	BlockTips *event.Event[*BlockTipsEvent]
}

func newCollectionEvents() (new *CollectionEvents) {
	return &CollectionEvents{
		AnalysisOutboundBytes: event.New[*AnalysisOutboundBytesEvent](),
		CPUUsage:              event.New[*CPUUsageEvent](),
		MemUsage:              event.New[*MemUsageEvent](),
		MeshTimeSynced:      event.New[*MeshTimeSyncedEvent](),
		ValueTips:             event.New[*ValueTipsEvent](),
		BlockTips:             event.New[*BlockTipsEvent](),
	}
}

type AnalysisOutboundBytesEvent struct {
	AmountBytes uint64
}

type CPUUsageEvent struct {
	CPUPercent float64
}

type MemUsageEvent struct {
	MemAllocBytes uint64
}

type MeshTimeSyncedEvent struct{}

type ValueTipsEvent struct{}

type BlockTipsEvent struct{}
