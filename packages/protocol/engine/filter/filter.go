package filter

import (
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/protocol/models"
)

type Filter interface {
	Events() *Events

	// ProcessReceivedBlock processes block from the given source.
	ProcessReceivedBlock(block *models.Block, source identity.ID)

	module.Interface
}
