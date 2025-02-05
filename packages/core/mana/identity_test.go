package mana

import (
	"testing"

	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
)

func TestIDFromStr(t *testing.T) {
	_identity := identity.GenerateIdentity()
	idBytes, err := _identity.ID().Bytes()
	if err != nil {
		t.Fatalf("Error converting NodeID to bytes: %v", err)
	}
	ID, err := IDFromStr(base58.Encode(idBytes))
	assert.NoError(t, err)
	assert.Equal(t, _identity.ID(), ID)
}

func TestIDFromPubKey(t *testing.T) {
	_identity := identity.GenerateIdentity()
	ID, err := IDFromPubKey(_identity.PublicKey().String())
	assert.NoError(t, err)
	assert.Equal(t, _identity.ID(), ID)
}
