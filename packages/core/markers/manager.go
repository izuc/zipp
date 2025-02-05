package markers

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/cockroachdb/errors"

	"github.com/izuc/zipp.foundation/core/generics/objectstorage"
	"github.com/izuc/zipp.foundation/core/generics/walker"
	"github.com/izuc/zipp.foundation/core/kvstore"
	"github.com/izuc/zipp.foundation/core/kvstore/mapdb"

	"github.com/izuc/zipp/packages/node/database"
)

// region Manager //////////////////////////////////////////////////////////////////////////////////////////////////////

// Manager is the managing entity for the Marker related business logic. It is stateful and automatically stores its
// state in a KVStore.
type Manager struct {
	Options *ManagerOptions

	sequenceStore          *objectstorage.ObjectStorage[*Sequence]
	sequenceIDCounter      SequenceID
	sequenceIDCounterMutex sync.Mutex
	shutdownOnce           sync.Once
}

// NewManager is the constructor of the Manager that takes a KVStore to persist its state.
func NewManager(options ...ManagerOption) (newManager *Manager) {
	newManager = &Manager{
		Options: DefaultManagerOptions.Apply(options...),
	}

	newManager.initSequenceIDCounter()
	newManager.initObjectStorage()

	return newManager
}

// InheritStructureDetails takes the StructureDetails of the referenced parents and returns new StructureDetails for the
// block that was just added to the DAG. It automatically creates a new Sequence and Index if necessary and returns an
// additional flag that indicates if a new Sequence was created.
// InheritStructureDetails inherits the structure details of the given parent StructureDetails.
func (m *Manager) InheritStructureDetails(referencedStructureDetails []*StructureDetails, increaseIndexCallback IncreaseIndexCallback) (inheritedStructureDetails *StructureDetails, newSequenceCreated bool) {
	inheritedStructureDetails = m.mergeParentStructureDetails(referencedStructureDetails)

	inheritedStructureDetails.SetPastMarkers(m.normalizeMarkers(inheritedStructureDetails.PastMarkers()))
	if inheritedStructureDetails.PastMarkers().Size() == 0 {
		inheritedStructureDetails.SetPastMarkers(NewMarkers(NewMarker(0, 0)))
	}

	assignedMarker, sequenceExtended := m.extendHighestAvailableSequence(inheritedStructureDetails.PastMarkers(), increaseIndexCallback)
	if !sequenceExtended {
		newSequenceCreated, assignedMarker = m.createSequenceIfNecessary(inheritedStructureDetails)
	}

	if !sequenceExtended && !newSequenceCreated {
		return inheritedStructureDetails, false
	}

	inheritedStructureDetails.SetIsPastMarker(true)
	inheritedStructureDetails.SetPastMarkerGap(0)
	inheritedStructureDetails.SetPastMarkers(NewMarkers(assignedMarker))

	return inheritedStructureDetails, newSequenceCreated
}

// Sequence retrieves a Sequence from the object storage.
func (m *Manager) Sequence(sequenceID SequenceID) *objectstorage.CachedObject[*Sequence] {
	return m.sequenceStore.Load(sequenceID.Bytes())
}

// Shutdown shuts down the Manager and persists its state.
func (m *Manager) Shutdown() {
	m.shutdownOnce.Do(func() {
		if err := m.Options.Store.Set(kvstore.Key("sequenceIDCounter"), m.sequenceIDCounter.Bytes()); err != nil {
			panic(err)
		}

		m.sequenceStore.Shutdown()
	})
}

// initSequenceIDCounter restores the sequenceIDCounter from the KVStore upon initialization.
func (m *Manager) initSequenceIDCounter() (self *Manager) {
	storedSequenceIDCounter, err := m.Options.Store.Get(kvstore.Key("sequenceIDCounter"))
	if err != nil && !errors.Is(err, kvstore.ErrKeyNotFound) {
		panic(err)
	}

	if storedSequenceIDCounter != nil {
		if err = m.sequenceIDCounter.FromBytes(storedSequenceIDCounter); err != nil {
			panic(err)
		}
	}

	return m
}

// initObjectStorage sets up the object storage for the Sequences.
func (m *Manager) initObjectStorage() (self *Manager) {
	m.sequenceStore = objectstorage.NewStructStorage[Sequence](objectstorage.NewStoreWithRealm(m.Options.Store, database.PrefixMarkers, 0), objectstorage.CacheTime(m.Options.CacheTime))

	if cachedSequence, stored := m.sequenceStore.StoreIfAbsent(NewSequence(0, NewMarkers())); stored {
		cachedSequence.Release()
	}

	return m
}

// mergeParentStructureDetails merges the information of a set of parent StructureDetails into a single StructureDetails
// object.
func (m *Manager) mergeParentStructureDetails(referencedStructureDetails []*StructureDetails) (mergedStructureDetails *StructureDetails) {
	mergedStructureDetails = NewStructureDetails()
	mergedStructureDetails.SetPastMarkerGap(math.MaxUint64)

	for _, referencedMarkerPair := range referencedStructureDetails {
		mergedStructureDetails.PastMarkers().Merge(referencedMarkerPair.PastMarkers())

		if referencedMarkerPair.PastMarkerGap() < mergedStructureDetails.PastMarkerGap() {
			mergedStructureDetails.SetPastMarkerGap(referencedMarkerPair.PastMarkerGap())
		}

		if referencedMarkerPair.Rank() > mergedStructureDetails.Rank() {
			mergedStructureDetails.SetRank(referencedMarkerPair.Rank())
		}
	}

	mergedStructureDetails.SetPastMarkerGap(mergedStructureDetails.PastMarkerGap() + 1)
	mergedStructureDetails.SetRank(mergedStructureDetails.Rank() + 1)

	return mergedStructureDetails
}

// normalizeMarkers takes a set of Markers and removes each Marker that is already referenced by another Marker in the
// same set (the remaining Markers are the "most special" Markers that reference all Markers in the set grouped by the
// rank of their corresponding Sequence). In addition, the method returns all SequenceIDs of the Markers that were not
// referenced by any of the Markers (the tips of the Sequence DAG).
func (m *Manager) normalizeMarkers(markers *Markers) (normalizedMarkers *Markers) {
	normalizedMarkers = markers.Clone()

	normalizeWalker := walker.New[Marker]()
	markers.ForEach(func(sequenceID SequenceID, index Index) bool {
		normalizeWalker.Push(NewMarker(sequenceID, index))

		return true
	})

	seenMarkers := NewMarkers()
	for i := 0; normalizeWalker.HasNext(); i++ {
		currentMarker := normalizeWalker.Next()

		if i >= markers.Size() {
			if added, updated := seenMarkers.Set(currentMarker.SequenceID(), currentMarker.Index()); !added && !updated {
				continue
			}

			index, exists := normalizedMarkers.Get(currentMarker.SequenceID())
			if exists {
				if index > currentMarker.Index() {
					continue
				}

				normalizedMarkers.Delete(currentMarker.SequenceID())
			}
		}

		if !m.Sequence(currentMarker.SequenceID()).Consume(func(sequence *Sequence) {
			sequence.ReferencedMarkers(currentMarker.Index()).ForEach(func(referencedSequenceID SequenceID, referencedIndex Index) bool {
				normalizeWalker.Push(NewMarker(referencedSequenceID, referencedIndex))

				return true
			})
		}) {
			panic(fmt.Sprintf("failed to load Sequence with %s", currentMarker.SequenceID()))
		}
	}

	return normalizedMarkers
}

// extendHighestAvailableSequence is an internal utility function that tries to extend the referenced Sequences in
// descending order. It returns the newly assigned Marker and a boolean value that indicates if one of the referenced
// Sequences could be extended.
func (m *Manager) extendHighestAvailableSequence(referencedPastMarkers *Markers, increaseIndexCallback IncreaseIndexCallback) (marker Marker, extended bool) {
	referencedPastMarkers.ForEachSorted(func(sequenceID SequenceID, index Index) bool {
		m.Sequence(sequenceID).Consume(func(sequence *Sequence) {
			if newIndex, remainingReferencedPastMarkers, sequenceExtended := sequence.TryExtend(referencedPastMarkers, increaseIndexCallback); sequenceExtended {
				extended = sequenceExtended
				marker = NewMarker(sequenceID, newIndex)

				m.registerReferencingMarker(remainingReferencedPastMarkers, marker)
			}
		})

		return !extended
	})

	return
}

// createSequenceIfNecessary is an internal utility function that creates a new Sequence if the distance to the last
// past Marker is higher or equal than the configured threshold and returns the first Marker in that Sequence.
func (m *Manager) createSequenceIfNecessary(structureDetails *StructureDetails) (created bool, firstMarker Marker) {
	if structureDetails.PastMarkerGap() < m.Options.MaxPastMarkerDistance {
		return
	}

	m.sequenceIDCounterMutex.Lock()
	m.sequenceIDCounter++
	newSequence := NewSequence(m.sequenceIDCounter, structureDetails.PastMarkers())
	m.sequenceIDCounterMutex.Unlock()

	m.sequenceStore.Store(newSequence).Release()

	firstMarker = NewMarker(newSequence.ID(), newSequence.LowestIndex())

	m.registerReferencingMarker(structureDetails.PastMarkers(), firstMarker)

	return true, firstMarker
}

// laterMarkersReferenceEarlierMarkers is an internal utility function that returns true if the later Markers reference
// the earlier Markers. If requireBiggerMarkers is false then a Marker with an equal Index is considered to be a valid
// reference.
func (m *Manager) laterMarkersReferenceEarlierMarkers(laterMarkers, earlierMarkers *Markers, requireBiggerMarkers bool) (result bool) {
	referenceWalker := walker.New[Marker]()
	laterMarkers.ForEach(func(sequenceID SequenceID, index Index) bool {
		referenceWalker.Push(NewMarker(sequenceID, index))
		return true
	})

	seenMarkers := NewMarkers()
	for i := 0; referenceWalker.HasNext(); i++ {
		laterMarker := referenceWalker.Next()
		if added, updated := seenMarkers.Set(laterMarker.SequenceID(), laterMarker.Index()); !added && !updated {
			continue
		}

		isInitialLaterMarker := i < laterMarkers.Size()
		if m.laterMarkerDirectlyReferencesEarlierMarkers(laterMarker, earlierMarkers, isInitialLaterMarker && requireBiggerMarkers) {
			return true
		}

		m.Sequence(laterMarker.SequenceID()).Consume(func(sequence *Sequence) {
			sequence.ReferencedMarkers(laterMarker.Index()).ForEach(func(referencedSequenceID SequenceID, referencedIndex Index) bool {
				referenceWalker.Push(NewMarker(referencedSequenceID, referencedIndex))
				return true
			})
		})
	}

	return false
}

// laterMarkerDirectlyReferencesEarlierMarkers returns true if the later Marker directly references the earlier Markers.
func (m *Manager) laterMarkerDirectlyReferencesEarlierMarkers(laterMarker Marker, earlierMarkers *Markers, requireBiggerMarkers bool) bool {
	earlierMarkersLowestIndex := earlierMarkers.LowestIndex()
	if requireBiggerMarkers {
		earlierMarkersLowestIndex++
	}

	if laterMarker.Index() < earlierMarkersLowestIndex {
		return false
	}

	earlierIndex, sequenceExists := earlierMarkers.Get(laterMarker.SequenceID())
	if !sequenceExists {
		return false
	}

	if requireBiggerMarkers {
		earlierIndex++
	}

	return laterMarker.Index() >= earlierIndex
}

// registerReferencingMarker is an internal utility function that adds a referencing Marker to the internal data
// structure.
func (m *Manager) registerReferencingMarker(referencedPastMarkers *Markers, marker Marker) {
	referencedPastMarkers.ForEach(func(sequenceID SequenceID, index Index) bool {
		m.Sequence(sequenceID).Consume(func(sequence *Sequence) {
			sequence.AddReferencingMarker(index, marker)
		})

		return true
	})
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region ManagerOptions ///////////////////////////////////////////////////////////////////////////////////////////////

// ManagerOption represents the return type of optional parameters that can be handed into the constructor of the
// Manager to configure its behavior.
type ManagerOption func(options *ManagerOptions)

// ManagerOptions is a container for all configurable parameters of the Manager.
type ManagerOptions struct {
	// Store is a parameter for the Manager that allows to specify which storage layer is supposed to be used to persist
	// data.
	Store kvstore.KVStore

	// CacheTime is a parameter for the Manager that allows to specify how long objects should be cached in the object
	// storage.
	CacheTime time.Duration

	// MaxPastMarkerDistance is a parameter for the Manager that allows to specify how many consecutive blocks are
	// allowed to not receive a new PastMaster before we create a new Sequence.
	MaxPastMarkerDistance uint64
}

// Apply applies the given options to the ManagerOptions object.
func (m *ManagerOptions) Apply(options ...ManagerOption) (managerOptions *ManagerOptions) {
	managerOptions = &ManagerOptions{
		Store:                 DefaultManagerOptions.Store,
		CacheTime:             DefaultManagerOptions.CacheTime,
		MaxPastMarkerDistance: DefaultManagerOptions.MaxPastMarkerDistance,
	}

	for _, option := range options {
		option(managerOptions)
	}
	return managerOptions
}

// DefaultManagerOptions defines the default options for the Manager.
var DefaultManagerOptions = &ManagerOptions{
	Store:                 mapdb.NewMapDB(),
	CacheTime:             30 * time.Second,
	MaxPastMarkerDistance: 30,
}

// WithStore is an option for the Manager that allows to specify which storage layer is supposed to be used to persist
// data.
func WithStore(store kvstore.KVStore) ManagerOption {
	return func(options *ManagerOptions) {
		options.Store = store
	}
}

// WithCacheTime is an option for the Manager that allows to specify how long objects should be cached in the object
// storage.
func WithCacheTime(cacheTime time.Duration) ManagerOption {
	return func(options *ManagerOptions) {
		options.CacheTime = cacheTime
	}
}

// WithMaxPastMarkerDistance is an Option for the Manager that allows to specify how many consecutive blocks are
// allowed to not receive a new PastMaster before we create a new Sequence.
func WithMaxPastMarkerDistance(distance uint64) ManagerOption {
	return func(options *ManagerOptions) {
		options.MaxPastMarkerDistance = distance
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
