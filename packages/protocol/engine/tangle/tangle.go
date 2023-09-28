package tangle

import (
	"github.com/izuc/zipp.foundation/runtime/module"
	"github.com/izuc/zipp/packages/protocol/engine/tangle/blockdag"
	"github.com/izuc/zipp/packages/protocol/engine/tangle/booker"
)

type Tangle interface {
	Events() *Events

	Booker() booker.Booker

	BlockDAG() blockdag.BlockDAG

	module.Interface
}
