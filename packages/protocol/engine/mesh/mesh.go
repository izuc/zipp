package mesh

import (
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/mesh/booker"
)

type Mesh interface {
	Events() *Events

	Booker() booker.Booker

	BlockDAG() blockdag.BlockDAG

	module.Interface
}
