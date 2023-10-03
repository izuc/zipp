package mesh_old

import (
	"github.com/izuc/zipp/packages/core/consensus"
)

// region OMVConsensusManager /////////////////////////////////////////////////////////////////////////////////////////////

// OMVConsensusManager is the component in charge of forming opinions about conflicts.
type OMVConsensusManager struct {
	consensus.Mechanism
}

// NewOMVConsensusManager returns a new Mechanism.
func NewOMVConsensusManager(omvConsensusMechanism consensus.Mechanism) *OMVConsensusManager {
	return &OMVConsensusManager{
		Mechanism: omvConsensusMechanism,
	}
}

// endregion ///////////////////////////////////////////////////////////////////////////////////////////////////////////
