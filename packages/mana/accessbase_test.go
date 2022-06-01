package mana

import (
	"math"
	"testing"
	"time"

	"github.com/iotaledger/hive.go/identity"
	"github.com/stretchr/testify/assert"
)

var delta = 0.001

func TestUpdateBM2(t *testing.T) {
	t.Run("CASE: Zero values", func(t *testing.T) {
		bm := AccessBaseMana{}

		// 0 initial values, timely update should not change anything
		bm.updateBM2(time.Hour)
		assert.Equal(t, 0.0, bm.BaseValue())
	})

	t.Run("CASE: Batch update", func(t *testing.T) {
		bm := AccessBaseMana{}

		// pledge BM2 at t = o
		bm.M.BaseValue = 1.0
		bm.updateBM2(time.Hour * 6)
		assert.InDelta(t, 0.5, bm.BaseValue(), delta)
	})

	t.Run("CASE: Incremental update", func(t *testing.T) {
		bm := AccessBaseMana{}

		// pledge BM2 at t = o
		bm.M.BaseValue = 1.0
		// with emaCoeff1 = 0.00003209, half value should be reached within 6 hours
		for i := 0; i < 6; i++ {
			bm.updateBM2(time.Hour)
		}
		assert.InDelta(t, 0.5, bm.BaseValue(), delta)
	})
}

func TestUpdateEBM2CoeffEqual(t *testing.T) {
	t.Run("CASE: Zero values", func(t *testing.T) {
		bm := AccessBaseMana{}

		// 0 initial values, timely update should not change anything
		bm.updateEBM2(time.Hour)
		assert.Equal(t, 0.0, bm.M.EffectiveValue)
	})

	t.Run("CASE: Batch and incremental update", func(t *testing.T) {
		bmBatch := AccessBaseMana{}

		// first, let's calculate once on a 6 hour span
		// pledge BM2 at t = o
		bmBatch.M.BaseValue = 1.0
		// updateEBM2 relies on an update baseMana2 value
		bmBatch.updateBM2(time.Hour * 6)
		bmBatch.updateEBM2(time.Hour * 6)

		bmInc := AccessBaseMana{}
		// second, let's calculate the same but every hour
		// pledge BM2 at t = o
		bmInc.M.BaseValue = 1.0
		// with emaCoeff1 = 0.00003209, half value should be reached within 6 hours
		for i := 0; i < 6; i++ {
			// updateEBM2 relies on an update baseMana2 value
			bmInc.updateBM2(time.Hour)
			bmInc.updateEBM2(time.Hour)
		}

		// compare results of the two calculations
		assert.Equal(t, true, math.Abs(bmBatch.M.EffectiveValue-bmInc.M.EffectiveValue) < delta)
	})

	t.Run("CASE: Large durations BM2", func(t *testing.T) {
		bmBatch := AccessBaseMana{}

		// first, let's calculate once on a 6 hour span
		// pledge BM2 at t = o
		bmBatch.M.BaseValue = 1.0
		// updateEBM2 relies on an update baseMana2 value
		minTime := time.Unix(-2208988800, 0) // Jan 1, 1900
		maxTime := minTime.Add(1<<63 - 1)

		bmBatch.updateBM2(maxTime.Sub(minTime))

		assert.False(t, math.IsNaN(bmBatch.M.BaseValue))
		assert.False(t, math.IsInf(bmBatch.M.BaseValue, 0))
		assert.Equal(t, 0.0, bmBatch.M.BaseValue)
	})

	t.Run("CASE: Large durations EBM2 Decay==emaCoeff2", func(t *testing.T) {
		bmBatch := AccessBaseMana{}
		// pledge BM2 at t = o
		bmBatch.M.BaseValue = 1.0
		bmBatch.M.EffectiveValue = 1.0

		// updateEBM2 relies on an update baseMana2 value
		minTime := time.Unix(-2208988800, 0) // Jan 1, 1900
		maxTime := minTime.Add(1<<63 - 1)

		bmBatch.updateBM2(maxTime.Sub(minTime))
		bmBatch.updateEBM2(maxTime.Sub(minTime))
		assert.False(t, math.IsNaN(bmBatch.M.BaseValue))
		assert.False(t, math.IsNaN(bmBatch.M.EffectiveValue))
		assert.False(t, math.IsInf(bmBatch.M.BaseValue, 0))
		assert.False(t, math.IsInf(bmBatch.M.EffectiveValue, 0))
		assert.Equal(t, 0.0, bmBatch.M.BaseValue)
		assert.Equal(t, 0.0, bmBatch.M.EffectiveValue)
	})
	t.Run("CASE: Large durations EBM2 Decay!=emaCoeff2", func(t *testing.T) {
		bmBatch := AccessBaseMana{}
		SetCoefficients(0.00003209, 0.0057762265, 0.00003209)
		// pledge BM2 at t = o
		bmBatch.M.BaseValue = 1.0
		bmBatch.M.EffectiveValue = 1.0

		// updateEBM2 relies on an update baseMana2 value
		minTime := time.Unix(-2208988800, 0) // Jan 1, 1900
		maxTime := minTime.Add(1<<63 - 1)

		bmBatch.updateBM2(maxTime.Sub(minTime))
		bmBatch.updateEBM2(maxTime.Sub(minTime))

		assert.False(t, math.IsNaN(bmBatch.M.BaseValue))
		assert.False(t, math.IsNaN(bmBatch.M.EffectiveValue))
		assert.False(t, math.IsInf(bmBatch.M.BaseValue, 0))
		assert.False(t, math.IsInf(bmBatch.M.EffectiveValue, 0))
		assert.Equal(t, 0.0, bmBatch.BaseValue())
		assert.Equal(t, 0.0, bmBatch.EffectiveValue())
		// re-set the default values so that other tests pass
		SetCoefficients(0.00003209, 0.00003209, 0.00003209)
	})
}

func TestUpdateTimeInPast_Access(t *testing.T) {
	baseTime := time.Now()
	bm := NewAccessBaseMana(1.0, 0.0, baseTime)
	pastTime := baseTime.Add(time.Hour * -1)
	err := bm.update(pastTime)
	assert.Error(t, err)
	assert.Equal(t, ErrAlreadyUpdated, err)
}

func TestUpdate_Access(t *testing.T) {
	baseTime := time.Now()
	bm := NewAccessBaseMana(1.0, 0.0, baseTime)

	updateTime := baseTime.Add(time.Hour * 6)

	err := bm.update(updateTime)
	assert.NoError(t, err)
	// values are only valid for default coefficients of 0.00003209 and t = 6 hours
	assert.InDelta(t, 0.5, bm.BaseValue(), delta)
	assert.InDelta(t, 0.346573, bm.EffectiveValue(), delta)
	assert.Equal(t, updateTime, bm.LastUpdate())
}

func TestRevoke_Access(t *testing.T) {
	baseTime := time.Now()
	bm := NewAccessBaseMana(1.0, 0.0, baseTime)

	assert.Panics(t, func() {
		_ = bm.revoke(1.0)
	})
}

func TestPledgeRegularOldFunds_Access(t *testing.T) {
	baseTime := time.Now()
	bm := NewAccessBaseMana(1.0, 0.0, baseTime)

	// transaction pledges mana at t=6 hours with 3 inputs.
	txInfo := &TxInfo{
		TimeStamp:    baseTime.Add(time.Hour * 6),
		TotalBalance: 10.0,
		PledgeID:     map[Type]identity.ID{}, // don't care
		InputInfos: []InputInfo{
			{
				// funds have been sitting here for couple days...
				TimeStamp: baseTime.Add(time.Hour * -200),
				Amount:    5.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
			{
				// funds have been sitting here for couple days...
				TimeStamp: baseTime.Add(time.Hour * -200),
				Amount:    3.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
			{
				// funds have been sitting here for couple days...
				TimeStamp: baseTime.Add(time.Hour * -200),
				Amount:    2.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
		},
	}

	bm2Pledged := bm.pledge(txInfo)

	assert.InDelta(t, 10.0, bm2Pledged, delta)
	// half of the original BM2 degraded away in 6 hours
	assert.InDelta(t, 10.5, bm.BaseValue(), delta)
}

func TestPledgeRegularHalfOldFunds_Access(t *testing.T) {
	baseTime := time.Now()
	bm := NewAccessBaseMana(1.0, 0.0, baseTime)

	// transaction pledges mana at t=6 hours with 3 inputs.
	txInfo := &TxInfo{
		TimeStamp:    baseTime.Add(time.Hour * 6),
		TotalBalance: 10.0,
		PledgeID:     map[Type]identity.ID{}, // don't care
		InputInfos: []InputInfo{
			{
				// funds have been sitting here for 6 hours
				TimeStamp: baseTime,
				Amount:    5.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
			{
				// funds have been sitting here for 6 hours
				TimeStamp: baseTime,
				Amount:    3.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
			{
				// funds have been sitting here for 6 hours
				TimeStamp: baseTime,
				Amount:    2.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
		},
	}

	bm2Pledged := bm.pledge(txInfo)

	assert.InDelta(t, 5.0, bm2Pledged, delta)
	// half of the original BM2 degraded away in 6 hours
	assert.InDelta(t, 5.5, bm.BaseValue(), delta)
}

func TestPledgePastOldFunds_Access(t *testing.T) {
	baseTime := time.Now()
	bm := NewAccessBaseMana(0, 0.0, baseTime.Add(time.Hour*6))

	// transaction pledges mana at t=0 hours with 3 inputs.
	txInfo := &TxInfo{
		TimeStamp:    baseTime,
		TotalBalance: 10.0,
		PledgeID:     map[Type]identity.ID{}, // don't care
		InputInfos: []InputInfo{
			{
				// funds have been sitting here for couple days...
				TimeStamp: baseTime.Add(time.Hour * -200),
				Amount:    5.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
			{
				// funds have been sitting here for couple days...
				TimeStamp: baseTime.Add(time.Hour * -200),
				Amount:    3.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
			{
				// funds have been sitting here for couple days...
				TimeStamp: baseTime.Add(time.Hour * -200),
				Amount:    2.0,
				PledgeID:  map[Type]identity.ID{}, // don't care
			},
		},
	}

	bm2Pledged := bm.pledge(txInfo)

	// pledged at t=0, half of input amount is added to bm2
	assert.InDelta(t, 5.0, bm2Pledged, delta)
	// half of the original BM2 degraded away in 6 hours
	// valid EBM2 at t=6 hours, after pledging 10 BM2 at t=0
	assert.InDelta(t, 3.465731, bm.EffectiveValue(), delta)
}
