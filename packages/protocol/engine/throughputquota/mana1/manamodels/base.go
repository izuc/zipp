package manamodels

import (
	"time"

	"github.com/izuc/zipp.foundation/crypto/identity"
)

// ManaRetrievalFunc returns the mana value of a node with default weights.
type ManaRetrievalFunc func(identity.ID, ...time.Time) (int64, time.Time, error)
