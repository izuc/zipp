package mana

import (
	"math"
	"testing"
	"time"

	"github.com/izuc/zipp.foundation/core/identity"

	"github.com/izuc/zipp.foundation/core/marshalutil"
	"github.com/stretchr/testify/assert"

	"github.com/izuc/zipp/packages/core/ledger/utxo"
)

func TestPersistableEvent_Bytes(t *testing.T) {
	ev := new(PersistableEvent)
	marshalUtil := marshalutil.New()
	marshalUtil.WriteByte(ev.Type)
	marshalUtil.WriteByte(byte(ev.ManaType))

	nodeIDBytes, err := ev.NodeID.Bytes()
	if err != nil {
		t.Fatal("Failed to get bytes from NodeID: ", err)
	}
	marshalUtil.WriteBytes(nodeIDBytes)

	marshalUtil.WriteTime(ev.Time)
	marshalUtil.WriteBytes(ev.TransactionID.Bytes())
	marshalUtil.WriteUint64(math.Float64bits(ev.Amount))

	// Directly use the result for InputID since there's no error to check
	marshalUtil.WriteBytes(ev.InputID.Bytes())

	bytes := marshalUtil.Bytes()
	assert.Equal(t, bytes, ev.Bytes(), "should be equal")
}

func TestPersistableEvent_ObjectStorageKey(t *testing.T) {
	ev := new(PersistableEvent)
	key := ev.Bytes()
	assert.Equal(t, key, ev.ObjectStorageKey(), "should be equal")
}

func TestPersistableEvent_ObjectStorageValue(t *testing.T) {
	ev := new(PersistableEvent)
	val := ev.ObjectStorageValue()
	assert.Equal(t, ev.Bytes(), val, "should be equal")
}

func TestPersistableEvent_FromBytes(t *testing.T) {
	ev := &PersistableEvent{
		Type:          EventTypePledge,
		NodeID:        identity.ID{},
		Amount:        100,
		Time:          time.Now(),
		ManaType:      ConsensusMana,
		TransactionID: utxo.TransactionID{},
	}
	ev1, err := new(PersistableEvent).FromBytes(ev.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, ev.Bytes(), ev1.Bytes(), "should be equal")
}
