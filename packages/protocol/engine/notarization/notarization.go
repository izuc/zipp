package notarization

import (
	"io"

	"github.com/izuc/zipp.foundation/ads"
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/core/traits"
	"github.com/izuc/zipp/packages/protocol/models"
)

type Notarization interface {
	Events() *Events

	Attestations() Attestations

	// IsFullyCommitted returns if notarization finished committing all pending slots up to the current acceptance time.
	IsFullyCommitted() bool

	NotarizeAcceptedBlock(block *models.Block) (err error)

	NotarizeOrphanedBlock(block *models.Block) (err error)

	Import(reader io.ReadSeeker) (err error)

	Export(writer io.WriteSeeker, targetSlot slot.Index) (err error)

	PerformLocked(perform func(m Notarization))

	module.Interface
}

type Attestations interface {
	Get(index slot.Index) (attestations *ads.Map[identity.ID, Attestation, *identity.ID, *Attestation], err error)

	traits.Committable
	module.Interface
}
