package storage

import "strconv"

// maxEpochsToKeep is the number of epochs we want to keep delegations in memory for data node.
const maxEpochsToKeep = 30

// clearOldEpochsDelegations makes sure we only keep as many as <maxEpochsToKeep> epoch entries.
func clearOldEpochsDelegations(epochSeq string, minEpoch *uint64, cleanup func(string)) {
	if minEpoch == nil {
		return
	}

	epochSeqUint, err := strconv.ParseUint(epochSeq, 10, 64)
	if err != nil {
		return
	}
	// if we see an epoch younger than we've seen before - update the min epoch
	if epochSeqUint <= *minEpoch {
		*minEpoch = epochSeqUint
	}
	// if we haven't seen yet <maxEpochsToKeep> or we have no more than the required number of epochs - we don't have anything to do here
	if epochSeqUint < maxEpochsToKeep || *minEpoch >= (epochSeqUint-maxEpochsToKeep+1) {
		return
	}

	// cleanup enough epochs such that we have at most <maxEpochsToKeep> epochs
	for i := *minEpoch; i < (epochSeqUint - maxEpochsToKeep + 1); i++ {
		cleanup(strconv.FormatUint(i, 10))
	}
	*minEpoch = epochSeqUint - maxEpochsToKeep + 1
}
