package inmemorymesh

import (
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp.foundation/runtime/options"
	"github.com/izuc/zipp/packages/protocol/engine"
	"github.com/izuc/zipp/packages/protocol/engine/mesh"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag/inmemoryblockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker/markerbooker"
)

// region Mesh ///////////////////////////////////////////////////////////////////////////////////////////////////////

// Mesh is a conflict free replicated data type that allows users to issue their own Blocks with each Block casting
// virtual votes on existing conflicts.
type Mesh struct {
	events *mesh.Events

	blockDAG blockdag.BlockDAG
	booker   booker.Booker

	optsBlockDAGProvider module.Provider[*engine.Engine, blockdag.BlockDAG]
	optsBookerProvider   module.Provider[*engine.Engine, booker.Booker]

	module.Module
}

func NewProvider(opts ...options.Option[Mesh]) module.Provider[*engine.Engine, mesh.Mesh] {
	return module.Provide(func(e *engine.Engine) mesh.Mesh {
		return options.Apply(&Mesh{
			events: mesh.NewEvents(),

			optsBlockDAGProvider: inmemoryblockdag.NewProvider(),
			optsBookerProvider: markerbooker.NewProvider(
				markerbooker.WithSlotCutoffCallback(e.LastConfirmedSlot),
				markerbooker.WithSequenceCutoffCallback(e.FirstUnacceptedMarker),
			),
		},
			opts,
			func(t *Mesh) {
				t.blockDAG = t.optsBlockDAGProvider(e)
				t.booker = t.optsBookerProvider(e)

				t.events.BlockDAG.LinkTo(t.blockDAG.Events())
				t.events.Booker.LinkTo(t.booker.Events())

				e.HookConstructed(func() {
					e.Events.Mesh.LinkTo(t.events)
					t.TriggerInitialized()
				})
			},
			(*Mesh).TriggerConstructed,
		)
	})
}

var _ mesh.Mesh = new(Mesh)

func (t *Mesh) Events() *mesh.Events {
	return t.events
}

func (t *Mesh) Booker() booker.Booker {
	return t.booker
}

func (t *Mesh) BlockDAG() blockdag.BlockDAG {
	return t.blockDAG
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////

// region Options //////////////////////////////////////////////////////////////////////////////////////////////////////

// WithBlockDAGProvider returns an Option for the Mesh that allows to pass in a provider for the BlockDAG.
func WithBlockDAGProvider(provider module.Provider[*engine.Engine, blockdag.BlockDAG]) options.Option[Mesh] {
	return func(mesh *Mesh) {
		mesh.optsBlockDAGProvider = provider
	}
}

// WithBookerProvider returns an Option for the Mesh that allows to pass in a provider for the Booker.
func WithBookerProvider(provider module.Provider[*engine.Engine, booker.Booker]) options.Option[Mesh] {
	return func(mesh *Mesh) {
		mesh.optsBookerProvider = provider
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
