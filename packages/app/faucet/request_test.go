package faucet

import (
	"testing"
	"time"

	"github.com/izuc/zipp.foundation/core/crypto/ed25519"
	"github.com/izuc/zipp.foundation/core/generics/lo"
	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/izuc/zipp.foundation/core/types"
	"github.com/stretchr/testify/assert"

	"github.com/izuc/zipp/packages/core/epoch"
	"github.com/izuc/zipp/packages/core/ledger/vm/devnetvm"
	"github.com/izuc/zipp/packages/core/mesh_old"
	"github.com/izuc/zipp/packages/core/mesh_old/payload"
)

func TestRequest(t *testing.T) {
	keyPair := ed25519.GenerateKeyPair()
	address := devnetvm.NewED25519Address(keyPair.PublicKey)
	access, _ := identity.RandomIDInsecure()
	consensus, _ := identity.RandomIDInsecure()

	originalRequest := NewRequest(address, access, consensus, 0)

	clonedRequest, _, err := FromBytes(lo.PanicOnErr(originalRequest.Bytes()))
	if err != nil {
		panic(err)
	}
	assert.Equal(t, originalRequest.Address(), clonedRequest.Address())
	assert.Equal(t, originalRequest.AccessManaPledgeID(), clonedRequest.AccessManaPledgeID())
	assert.Equal(t, originalRequest.ConsensusManaPledgeID(), clonedRequest.ConsensusManaPledgeID())

	clonedRequest2, _, err := FromBytes(lo.PanicOnErr(clonedRequest.Bytes()))
	if err != nil {
		panic(err)
	}

	assert.Equal(t, originalRequest.Address(), clonedRequest2.Address())
}

func TestIsFaucetReq(t *testing.T) {
	keyPair := ed25519.GenerateKeyPair()
	address := devnetvm.NewED25519Address(keyPair.PublicKey)
	local := identity.NewLocalIdentity(keyPair.PublicKey, keyPair.PrivateKey)
	emptyID := identity.ID{}

	faucetRequest := NewRequest(address, emptyID, emptyID, 0)

	faucetBlk := mesh_old.NewBlock(
		map[mesh_old.ParentsType]mesh_old.BlockIDs{
			mesh_old.StrongParentType: {
				mesh_old.EmptyBlockID: types.Void,
			},
		},
		time.Now(),
		local.PublicKey(),
		0,
		faucetRequest,
		0,
		ed25519.EmptySignature,
		0,
		epoch.NewECRecord(0),
	)

	dataBlk := mesh_old.NewBlock(
		map[mesh_old.ParentsType]mesh_old.BlockIDs{
			mesh_old.StrongParentType: {
				mesh_old.EmptyBlockID: types.Void,
			},
		},
		time.Now(),
		local.PublicKey(),
		0,
		payload.NewGenericDataPayload([]byte("data")),
		0,
		ed25519.EmptySignature,
		0,
		epoch.NewECRecord(0),
	)

	assert.Equal(t, true, IsFaucetReq(faucetBlk))
	assert.Equal(t, false, IsFaucetReq(dataBlk))
}
