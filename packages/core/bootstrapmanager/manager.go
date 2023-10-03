package bootstrapmanager

import (
	"github.com/izuc/zipp.foundation/core/generics/event"

	"github.com/izuc/zipp/packages/core/notarization"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

// region Manager //////////////////////////////////////////////////////////////////////////////////////////////////////

type BootstrappedEvent struct{}

type Events struct {
	Bootstrapped *event.Event[*BootstrappedEvent]
}

// Manager is the bootstrap manager.
type Manager struct {
	Events              *Events
	mesh              *mesh_old.Mesh
	notarizationManager *notarization.Manager
}

// New creates and returns a new notarization manager.
func New(t *mesh_old.Mesh, notarizationManager *notarization.Manager) (new *Manager) {
	new = &Manager{
		mesh:              t,
		notarizationManager: notarizationManager,
		Events:              &Events{Bootstrapped: event.New[*BootstrappedEvent]()},
	}
	return new
}

func (m *Manager) Setup() {
	m.mesh.TimeManager.Events.Bootstrapped.Attach(event.NewClosure(func(_ *mesh_old.BootstrappedEvent) {
		if m.notarizationManager.Bootstrapped() {
			m.Events.Bootstrapped.Trigger(&BootstrappedEvent{})
		}
	}))
	m.notarizationManager.Events.Bootstrapped.Attach(event.NewClosure(func(_ *notarization.BootstrappedEvent) {
		if m.mesh.Bootstrapped() {
			m.Events.Bootstrapped.Trigger(&BootstrappedEvent{})
		}
	}))
}

// Bootstrapped returns bool indicating if the node is bootstrapped.
func (m *Manager) Bootstrapped() bool {
	return m.mesh.Bootstrapped() && m.notarizationManager.Bootstrapped()
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
