package votes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/iotaledger/goshimmer/packages/core/validator"
	"github.com/iotaledger/goshimmer/packages/protocol/engine/sybilprotection/pos"
)

func TestActiveNodes_Update(t *testing.T) {
	activityNodes := pos.NewActiveNodes(time.Now, pos.WithActivityWindow(time.Second))

	tf := NewTestFramework(t, WithActiveNodes(activityNodes))

	tf.CreateValidator("A", validator.WithWeight(1))
	tf.CreateValidator("B", validator.WithWeight(1))
	tf.CreateValidator("C", validator.WithWeight(1))

	activityNodes.MarkActive(tf.Validator("A"), time.Now())
	activityNodes.MarkActive(tf.Validator("B"), time.Now())
	activityNodes.MarkActive(tf.Validator("C"), time.Now())

	assert.EqualValues(t, 3, tf.ActiveNodes.Weight())

	assert.Eventually(t, func() bool {
		return tf.ActiveNodes.Weight() == 0
	}, time.Second*3, time.Millisecond*10)
}
