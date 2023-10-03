package mana

import (
	"math"
	"testing"
	"time"

	"github.com/izuc/zipp.foundation/core/generics/lo"
	"github.com/izuc/zipp.foundation/core/identity"
	"github.com/izuc/zipp.foundation/core/marshalutil"
	"github.com/stretchr/testify/assert"
)

func TestPersistableBaseMana_Bytes(t *testing.T) {
	p := newPersistableMana()
	marshalUtil := marshalutil.New()
	marshalUtil.WriteByte(byte(p.ManaType()))
	marshalUtil.WriteUint16(uint16(len(p.BaseValues())))
	for _, val := range p.BaseValues() {
		marshalUtil.WriteUint64(math.Float64bits(val))
	}
	marshalUtil.WriteUint16(uint16(len(p.EffectiveValues())))
	for _, val := range p.EffectiveValues() {
		marshalUtil.WriteUint64(math.Float64bits(val))
	}

	marshalUtil.WriteTime(p.LastUpdated())

	bytes := marshalUtil.Bytes()
	assert.Equal(t, bytes, lo.PanicOnErr(p.Bytes()), "should be equal")
}

func TestPersistableBaseMana_ObjectStorageKey(t *testing.T) {
	p := newPersistableMana()
	keyBytes, err := identity.ID{}.Bytes()
	if err != nil {
		t.Fatal("Failed to get bytes from identity.ID: ", err)
	}
	assert.Equal(t, keyBytes, p.ObjectStorageKey(), "should be equal")
}

func TestPersistableBaseMana_ObjectStorageValue(t *testing.T) {
	p := newPersistableMana()
	val := p.ObjectStorageValue()
	assert.Equal(t, lo.PanicOnErr(p.Bytes()), val, "should be equal")
}

func TestPersistableBaseMana_FromBytes(t *testing.T) {
	p1 := newPersistableMana()
	p2 := new(PersistableBaseMana)

	bytes := lo.PanicOnErr(p1.Bytes())
	_, err := p2.FromBytes(bytes)
	if err != nil {
		t.Fatal("FromBytes returned an error: ", err)
	}

	// Note: Depending on your requirements, you may also want to check the value of consumedBytes
	assert.Equal(t, lo.PanicOnErr(p1.Bytes()), lo.PanicOnErr(p2.Bytes()), "should be equal")
}

func newPersistableMana() *PersistableBaseMana {
	return NewPersistableBaseMana(identity.ID{}, ConsensusMana, []float64{1, 1}, []float64{1, 1}, time.Now())
}
