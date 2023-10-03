package mesh_old

import (
	"math/rand"
	"testing"

	"github.com/izuc/zipp.foundation/core/generics/lo"
	"github.com/stretchr/testify/assert"

	"github.com/izuc/zipp/packages/core/ledger/utxo"
)

func TestMarkerIndexConflictIDMapping_Serialization(t *testing.T) {
	m := NewMarkerIndexConflictIDMapping(1)
	txID := utxo.NewTransactionID([]byte("1"))
	txID.RegisterAlias("txID")
	m.SetConflictIDs(10, utxo.NewTransactionIDs(txID))

	restored := new(MarkerIndexConflictIDMapping)
	err := restored.FromBytes(lo.PanicOnErr(m.Bytes()))
	assert.NoError(t, err)
	assert.Equal(t, m.ConflictIDs(11), restored.ConflictIDs(11))
}

func TestStorage_StoreAttachment(t *testing.T) {
	mesh := NewTestMesh()
	defer mesh.Shutdown()

	transactionID := randomTransactionID()
	blockID := randomBlockID()
	cachedAttachment, stored := mesh.Storage.StoreAttachment(transactionID, blockID)
	cachedAttachment.Release()
	assert.True(t, stored)
	cachedAttachment, stored = mesh.Storage.StoreAttachment(transactionID, randomBlockID())
	assert.True(t, stored)
	cachedAttachment.Release()
	cachedAttachment, stored = mesh.Storage.StoreAttachment(transactionID, blockID)
	assert.False(t, stored)
	assert.Nil(t, cachedAttachment)
}

func TestStorage_Attachments(t *testing.T) {
	mesh := NewTestMesh()
	defer mesh.Shutdown()

	attachments := make(map[utxo.TransactionID]int)
	for i := 0; i < 2; i++ {
		transactionID := randomTransactionID()

		// for every tx, store random number of attachments.
		for j := 0; j < rand.Intn(5)+1; j++ {
			attachments[transactionID]++
			cachedAttachment, _ := mesh.Storage.StoreAttachment(transactionID, randomBlockID())
			cachedAttachment.Release()
		}
	}

	for transactionID := range attachments {
		cachedAttachments := mesh.Storage.Attachments(transactionID)
		assert.Equal(t, attachments[transactionID], len(cachedAttachments))
		for _, cachedAttachment := range cachedAttachments {
			cachedAttachment.Release()
		}
	}
}
