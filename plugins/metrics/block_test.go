package metrics

import (
	"testing"

	"github.com/magiconair/properties/assert"

	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/mesh_old/payload"
)

func TestBlockCountPerPayload(t *testing.T) {
	// it is empty initially
	assert.Equal(t, BlockCountSinceStartPerComponentGrafana()[Store], uint64(0))
	// simulate attaching 10 transaction payloads in 0s < t < 1s
	for i := 0; i < 10; i++ {
		increasePerComponentCounter(Store)
		increasePerPayloadCounter(devnetvm.TransactionType)
	}
	assert.Equal(t, BlockCountSinceStartPerComponentGrafana()[Store], uint64(10))
	assert.Equal(t, BlockCountSinceStartPerPayload(), map[payload.Type]uint64{devnetvm.TransactionType: 10})
}
