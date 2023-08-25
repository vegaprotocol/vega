// Copyright (c) 2023 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package referral

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

type runningVolumes struct {
	// maxPartyNotionalVolumeByQuantumPerEpoch limits the volume in quantum units
	// which is eligible each epoch for referral program mechanisms.
	maxPartyNotionalVolumeByQuantumPerEpoch *num.Uint

	// runningVolumesBySet tracks the running notional volume by referralSets.
	runningVolumesBySet map[types.ReferralSetID][]*notionalVolume
}

type notionalVolume struct {
	epoch uint64
	value *num.Uint
}

func (vs *runningVolumes) Add(epoch uint64, setID types.ReferralSetID, volume *num.Uint) {
	volumeToAdd := volume
	if volume.GT(vs.maxPartyNotionalVolumeByQuantumPerEpoch) {
		volumeToAdd = vs.maxPartyNotionalVolumeByQuantumPerEpoch
	}

	runningVolumeForSet, isTracked := vs.runningVolumesBySet[setID]
	if !isTracked {
		vs.runningVolumesBySet[setID] = []*notionalVolume{
			{
				epoch: epoch,
				value: volumeToAdd.Clone(),
			},
		}
		return
	}

	for _, notionalVolume := range runningVolumeForSet {
		if notionalVolume.epoch == epoch {
			notionalVolume.value.AddSum(volumeToAdd)
			return
		}
	}

	// If we end up here, it means the set is tracked but the epoch is not.
	vs.runningVolumesBySet[setID] = append(runningVolumeForSet, &notionalVolume{
		epoch: epoch,
		value: volumeToAdd.Clone(),
	})
}

func (vs *runningVolumes) RunningSetVolumeForWindow(setID types.ReferralSetID, window uint64) *num.Uint {
	runningVolumeSet, isTracked := vs.runningVolumesBySet[setID]
	if !isTracked {
		return num.UintZero()
	}

	trackedEpochsCount := uint64(len(runningVolumeSet))
	startIndex := uint64(0)
	if trackedEpochsCount > window {
		startIndex = trackedEpochsCount - window - 1
	}

	runningVolumeForWindow := num.UintZero()
	for i := startIndex; i < trackedEpochsCount; i++ {
		runningVolumeForWindow.AddSum(runningVolumeSet[i].value)
	}

	return runningVolumeForWindow
}

func newRunningVolumes() *runningVolumes {
	return &runningVolumes{
		maxPartyNotionalVolumeByQuantumPerEpoch: num.UintZero(),
		runningVolumesBySet:                     map[types.ReferralSetID][]*notionalVolume{},
	}
}
