package mana

import (
	"time"

	"github.com/iotaledger/hive.go/generics/model"
	"github.com/iotaledger/hive.go/identity"
)

// PersistableBaseMana represents a base mana vector that can be persisted.
type PersistableBaseMana struct {
	model.Storable[identity.ID, persistableBaseManaModel] `serix:"0"`
}

type persistableBaseManaModel struct {
	ManaType        Type      `serix:"0"`
	BaseValues      []float64 `serix:"1,lengthPrefixType=uint16"`
	EffectiveValues []float64 `serix:"2,lengthPrefixType=uint16"`
	LastUpdated     time.Time `serix:"3"`
}

func NewPersistableBaseMana(nodeID identity.ID, manaType Type, baseValues, effectiveValues []float64, lastUpdated time.Time) *PersistableBaseMana {
	persistableBaseMana := &PersistableBaseMana{
		model.NewStorable[identity.ID, persistableBaseManaModel](
			persistableBaseManaModel{
				ManaType:        manaType,
				BaseValues:      baseValues,
				EffectiveValues: effectiveValues,
				LastUpdated:     lastUpdated,
			},
		),
	}
	persistableBaseMana.SetID(nodeID)
	return persistableBaseMana
}

func (v *PersistableBaseMana) NodeID() identity.ID {
	return v.ID()
}

func (v *PersistableBaseMana) ManaType() Type {
	v.RLock()
	defer v.RUnlock()
	return v.M.ManaType
}

func (v *PersistableBaseMana) BaseValues() []float64 {
	v.RLock()
	defer v.RUnlock()
	return v.M.BaseValues
}

func (v *PersistableBaseMana) EffectiveValues() []float64 {
	v.RLock()
	defer v.RUnlock()
	return v.M.EffectiveValues
}

func (v *PersistableBaseMana) LastUpdated() time.Time {
	v.RLock()
	defer v.RUnlock()
	return v.M.LastUpdated
}
