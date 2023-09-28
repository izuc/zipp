package slotgadget

import (
	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/runtime/module"
)

type Gadget interface {
	Events() *Events

	LastConfirmedSlot() slot.Index

	module.Interface
}
