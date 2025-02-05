package mesh_old

import (
	"sync"
	"time"

	"github.com/izuc/zipp.foundation/core/crypto"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/timed"
)

// region Requester ////////////////////////////////////////////////////////////////////////////////////////////////////

// Requester takes care of requesting blocks.
type Requester struct {
	mesh              *Mesh
	timedExecutor     *timed.Executor
	scheduledRequests map[BlockID]*timed.ScheduledTask
	options           RequesterOptions
	Events            *RequesterEvents

	scheduledRequestsMutex sync.RWMutex
}

// NewRequester creates a new block requester.
func NewRequester(mesh *Mesh, optionalOptions ...RequesterOption) *Requester {
	requester := &Requester{
		mesh:              mesh,
		timedExecutor:     timed.NewExecutor(1),
		scheduledRequests: make(map[BlockID]*timed.ScheduledTask),
		options:           DefaultRequesterOptions.Apply(optionalOptions...),
		Events:            newRequesterEvents(),
	}

	// add requests for all missing blocks
	requester.scheduledRequestsMutex.Lock()
	defer requester.scheduledRequestsMutex.Unlock()

	for _, id := range mesh.Storage.MissingBlocks() {
		requester.scheduledRequests[id] = requester.timedExecutor.ExecuteAfter(requester.createReRequest(id, 0), requester.options.RetryInterval+time.Duration(crypto.Randomness.Float64()*float64(requester.options.RetryJitter)))
	}

	return requester
}

// Setup sets up the behavior of the component by making it attach to the relevant events of other components.
func (r *Requester) Setup() {
	r.mesh.Solidifier.Events.BlockMissing.Hook(event.NewClosure(func(event *BlockMissingEvent) {
		r.StartRequest(event.BlockID)
	}))
	r.mesh.Storage.Events.MissingBlockStored.Hook(event.NewClosure(func(event *MissingBlockStoredEvent) {
		r.StopRequest(event.BlockID)
	}))
}

// Shutdown shuts down the Requester.
func (r *Requester) Shutdown() {
	r.timedExecutor.Shutdown(timed.CancelPendingElements)
}

// StartRequest initiates a regular triggering of the StartRequest event until it has been stopped using StopRequest.
func (r *Requester) StartRequest(id BlockID) {
	r.scheduledRequestsMutex.Lock()

	// ignore already scheduled requests
	if _, exists := r.scheduledRequests[id]; exists {
		r.scheduledRequestsMutex.Unlock()
		return
	}

	// schedule the next request and trigger the event
	r.scheduledRequests[id] = r.timedExecutor.ExecuteAfter(r.createReRequest(id, 0), r.options.RetryInterval+time.Duration(crypto.Randomness.Float64()*float64(r.options.RetryJitter)))
	r.scheduledRequestsMutex.Unlock()

	r.Events.RequestStarted.Trigger(&RequestStartedEvent{id})
	r.Events.RequestIssued.Trigger(&RequestIssuedEvent{id})
}

// StopRequest stops requests for the given block to further happen.
func (r *Requester) StopRequest(id BlockID) {
	r.scheduledRequestsMutex.Lock()

	timer, ok := r.scheduledRequests[id]
	if !ok {
		r.scheduledRequestsMutex.Unlock()
		return
	}

	timer.Cancel()
	delete(r.scheduledRequests, id)
	r.scheduledRequestsMutex.Unlock()

	r.Events.RequestStopped.Trigger(&RequestStoppedEvent{id})
}

func (r *Requester) reRequest(id BlockID, count int) {
	r.Events.RequestIssued.Trigger(&RequestIssuedEvent{id})

	// as we schedule a request at most once per id we do not need to make the trigger and the re-schedule atomic
	r.scheduledRequestsMutex.Lock()
	defer r.scheduledRequestsMutex.Unlock()

	// reschedule, if the request has not been stopped in the meantime
	if _, exists := r.scheduledRequests[id]; exists {
		// increase the request counter
		count++

		// if we have requested too often => stop the requests
		if count > r.options.MaxRequestThreshold {
			delete(r.scheduledRequests, id)

			r.Events.RequestFailed.Trigger(&RequestFailedEvent{id})
			r.mesh.Storage.DeleteMissingBlock(id)

			return
		}

		r.scheduledRequests[id] = r.timedExecutor.ExecuteAfter(r.createReRequest(id, count), r.options.RetryInterval+time.Duration(crypto.Randomness.Float64()*float64(r.options.RetryJitter)))
		return
	}
}

// RequestQueueSize returns the number of scheduled block requests.
func (r *Requester) RequestQueueSize() int {
	r.scheduledRequestsMutex.RLock()
	defer r.scheduledRequestsMutex.RUnlock()
	return len(r.scheduledRequests)
}

func (r *Requester) createReRequest(blkID BlockID, count int) func() {
	return func() { r.reRequest(blkID, count) }
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region RequesterOptions /////////////////////////////////////////////////////////////////////////////////////////////

// DefaultRequesterOptions defines the default options that are used when creating Requester instances.
var DefaultRequesterOptions = &RequesterOptions{
	RetryInterval:       10 * time.Second,
	RetryJitter:         10 * time.Second,
	MaxRequestThreshold: 500,
}

// RequesterOptions holds options for a block requester.
type RequesterOptions struct {
	// RetryInterval represents an option which defines in which intervals the Requester will try to ask for missing
	// blocks.
	RetryInterval time.Duration

	// RetryJitter defines how much the RetryInterval should be randomized, so that the nodes don't always send blocks
	// at exactly the same interval.
	RetryJitter time.Duration

	// MaxRequestThreshold represents an option which defines how often the Requester should try to request blocks
	// before canceling the request
	MaxRequestThreshold int
}

// Apply applies the optional Options to the RequesterOptions.
func (r RequesterOptions) Apply(optionalOptions ...RequesterOption) (updatedOptions RequesterOptions) {
	updatedOptions = r
	for _, optionalOption := range optionalOptions {
		optionalOption(&updatedOptions)
	}

	return updatedOptions
}

// RequesterOption is a function which inits an option.
type RequesterOption func(*RequesterOptions)

// RetryInterval creates an option which sets the retry interval to the given value.
func RetryInterval(interval time.Duration) RequesterOption {
	return func(args *RequesterOptions) {
		args.RetryInterval = interval
	}
}

// RetryJitter creates an option which sets the retry jitter to the given value.
func RetryJitter(retryJitter time.Duration) RequesterOption {
	return func(args *RequesterOptions) {
		args.RetryJitter = retryJitter
	}
}

// MaxRequestThreshold creates an option which defines how often the Requester should try to request blocks before
// canceling the request.
func MaxRequestThreshold(maxRequestThreshold int) RequesterOption {
	return func(args *RequesterOptions) {
		args.MaxRequestThreshold = maxRequestThreshold
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
