package blocklayer

import (
	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp.foundation/core/generics/event"
	"github.com/izuc/zipp.foundation/core/identity"

	"github.com/izuc/zipp/packages/core/consensus/acceptance"
	"github.com/izuc/zipp/packages/core/mesh_old"
)

var acceptanceGadget *acceptance.Gadget

// AcceptanceGadget is the finality gadget instance.
func AcceptanceGadget() *acceptance.Gadget {
	return acceptanceGadget
}

func configureFinality() {
	deps.Mesh.ApprovalWeightManager.Events.MarkerWeightChanged.Attach(event.NewClosure(func(e *mesh_old.MarkerWeightChangedEvent) {
		if err := acceptanceGadget.HandleMarker(e.Marker, e.Weight); err != nil {
			Plugin.LogError(err)
		}
	}))
	deps.Mesh.ApprovalWeightManager.Events.ConflictWeightChanged.Attach(event.NewClosure(func(e *mesh_old.ConflictWeightChangedEvent) {
		if err := acceptanceGadget.HandleConflict(e.ConflictID, e.Weight); err != nil {
			Plugin.LogError(err)
		}
	}))

	// we need to update the WeightProvider on confirmation
	acceptanceGadget.Events().BlockAccepted.Attach(event.NewClosure(func(event *mesh_old.BlockAcceptedEvent) {
		ei := epoch.IndexFromTime(event.Block.IssuingTime())
		deps.Mesh.WeightProvider.Update(ei, identity.NewID(event.Block.IssuerPublicKey()))
	}))
}
