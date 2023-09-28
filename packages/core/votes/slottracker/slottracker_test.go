package slottracker

import (
	"testing"

	"github.com/izuc/zipp.foundation/core/slot"
	"github.com/izuc/zipp.foundation/crypto/identity"
	"github.com/izuc/zipp.foundation/ds/advancedset"
	"github.com/izuc/zipp.foundation/lo"
	"github.com/izuc/zipp.foundation/runtime/debug"
)

func TestSlotTracker_TrackVotes(t *testing.T) {
	debug.SetEnabled(true)
	defer debug.SetEnabled(false)

	tf := NewDefaultTestFramework(t)

	tf.Votes.CreateValidator("A", 1)
	tf.Votes.CreateValidator("B", 1)

	expectedVoters := map[slot.Index]*advancedset.AdvancedSet[identity.ID]{
		1: tf.Votes.ValidatorsSet(),
		2: tf.Votes.ValidatorsSet(),
		3: tf.Votes.ValidatorsSet(),
		4: tf.Votes.ValidatorsSet(),
		5: tf.Votes.ValidatorsSet(),
		6: tf.Votes.ValidatorsSet(),
	}

	{
		tf.SlotTracker.TrackVotes(slot.Index(1), tf.Votes.Validator("A"), SlotVotePower{6})

		tf.ValidateSlotVoters(lo.MergeMaps(expectedVoters, map[slot.Index]*advancedset.AdvancedSet[identity.ID]{
			1: tf.Votes.ValidatorsSet("A"),
		}))
	}

	{
		tf.SlotTracker.TrackVotes(slot.Index(2), tf.Votes.Validator("A"), SlotVotePower{7})

		tf.ValidateSlotVoters(lo.MergeMaps(expectedVoters, map[slot.Index]*advancedset.AdvancedSet[identity.ID]{
			2: tf.Votes.ValidatorsSet("A"),
		}))
	}

	{
		tf.SlotTracker.TrackVotes(slot.Index(5), tf.Votes.Validator("A"), SlotVotePower{11})

		tf.ValidateSlotVoters(lo.MergeMaps(expectedVoters, map[slot.Index]*advancedset.AdvancedSet[identity.ID]{
			3: tf.Votes.ValidatorsSet("A"),
			4: tf.Votes.ValidatorsSet("A"),
			5: tf.Votes.ValidatorsSet("A"),
		}))
	}

	{
		tf.SlotTracker.TrackVotes(slot.Index(6), tf.Votes.Validator("B"), SlotVotePower{12})

		tf.ValidateSlotVoters(lo.MergeMaps(expectedVoters, map[slot.Index]*advancedset.AdvancedSet[identity.ID]{
			1: tf.Votes.ValidatorsSet("A", "B"),
			2: tf.Votes.ValidatorsSet("A", "B"),
			3: tf.Votes.ValidatorsSet("A", "B"),
			4: tf.Votes.ValidatorsSet("A", "B"),
			5: tf.Votes.ValidatorsSet("A", "B"),
			6: tf.Votes.ValidatorsSet("B"),
		}))
	}
}
