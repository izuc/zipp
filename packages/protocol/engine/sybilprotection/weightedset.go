package sybilprotection

import (
	"sync"

	"github.com/iotaledger/hive.go/core/generics/event"
	"github.com/iotaledger/hive.go/core/generics/set"
	"github.com/iotaledger/hive.go/core/identity"
)

type WeightedSet struct {
	Weights              *Weights
	weightUpdatesClosure *event.Closure[*WeightUpdatedEvent]
	members              *set.AdvancedSet[identity.ID]
	membersMutex         sync.RWMutex
	totalWeight          int64
	totalWeightMutex     sync.RWMutex
}

func NewWeightedSet(weights *Weights, optMembers ...identity.ID) (newWeightedSet *WeightedSet) {
	newWeightedSet = new(WeightedSet)
	newWeightedSet.Weights = weights
	newWeightedSet.weightUpdatesClosure = event.NewClosure(newWeightedSet.onWeightUpdated)
	newWeightedSet.members = set.NewAdvancedSet[identity.ID]()

	weights.Events.WeightUpdated.Attach(newWeightedSet.weightUpdatesClosure)

	for _, member := range optMembers {
		newWeightedSet.Add(member)
	}

	return
}

func (w *WeightedSet) Add(id identity.ID) (added bool) {
	w.Weights.mutex.RLock(id)
	defer w.Weights.mutex.RUnlock(id)

	w.membersMutex.Lock()
	defer w.membersMutex.Unlock()

	if added = w.members.Add(id); added {
		if weight, exists := w.Weights.weights.Get(id); exists {
			w.totalWeight += weight.Value
		}
	}

	return
}

func (w *WeightedSet) Delete(id identity.ID) (removed bool) {
	w.Weights.mutex.RLock(id)
	defer w.Weights.mutex.RUnlock(id)

	w.membersMutex.Lock()
	defer w.membersMutex.Unlock()

	if removed = w.members.Delete(id); removed {
		if weight, exists := w.Weights.weights.Get(id); exists {
			w.totalWeight -= weight.Value
		}
	}

	return
}

func (w *WeightedSet) Has(id identity.ID) (has bool) {
	w.membersMutex.RLock()
	defer w.membersMutex.RUnlock()

	return w.members.Has(id)
}

func (w *WeightedSet) ForEach(callback func(id identity.ID) error) (err error) {
	for _, member := range w.Slice() {
		if err = callback(member); err != nil {
			return
		}
	}

	return
}

func (w *WeightedSet) Weight() (totalWeight int64) {
	w.totalWeightMutex.RLock()
	defer w.totalWeightMutex.RUnlock()

	return w.totalWeight
}

func (w *WeightedSet) Slice() []identity.ID {
	w.membersMutex.RLock()
	defer w.membersMutex.RUnlock()

	return w.members.Slice()
}

func (w *WeightedSet) Detach() {
	w.Weights.Events.WeightUpdated.Detach(w.weightUpdatesClosure)
}

func (w *WeightedSet) onWeightUpdated(event *WeightUpdatedEvent) {
	if !w.members.Has(event.ID) {
		return
	}

	if event.OldWeight == nil {
		w.updateTotalWeight(event.NewWeight.Value)
	} else {
		w.updateTotalWeight(event.NewWeight.Value - event.OldWeight.Value)
	}
}

func (w *WeightedSet) updateTotalWeight(diff int64) {
	w.totalWeightMutex.Lock()
	defer w.totalWeightMutex.Unlock()

	w.totalWeight += diff
}
