package chainmanager

import (
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp/packages/core/commitment"
)

type Fork struct {
	Source       identity.ID
	Commitment   *commitment.Commitment
	ForkingPoint *commitment.Commitment
}
