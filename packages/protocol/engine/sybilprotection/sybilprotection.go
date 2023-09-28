package sybilprotection

import (
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/core/traits"
)

// SybilProtection is the minimal interface for the SybilProtection component of the ZIPP protocol.
type SybilProtection interface {
	// Weights returns the weights of identities in the SybilProtection.
	Weights() (weights *Weights)

	// Validators returns the set of online validators that is used to track acceptance.
	Validators() (validators *WeightedSet)

	// Committable is a trait that stores information about the latest commitment.
	traits.Committable

	// Interface embeds the required methods of the module.Interface.
	module.Interface
}
