package chat

import (
	"testing"

	"github.com/izuc/zipp.foundation/core/generics/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPayload(t *testing.T) {
	a := NewPayload("me", "you", "ciao")
	abytes := lo.PanicOnErr(a.Bytes())
	b := new(Payload)
	_, err := b.FromBytes(abytes)
	require.NoError(t, err)
	assert.Equal(t, a, b)
	// If needed, you can also use or check the value of consumedBytes.
}
