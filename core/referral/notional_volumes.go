// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

	notionalVolumeForEpoch := runningVolumeForSet[len(runningVolumeForSet)-1]
	if notionalVolumeForEpoch.epoch == epoch {
		notionalVolumeForEpoch.value.AddSum(volumeToAdd)
		return
	}

	// If we end up here, it means this set is tracked but this epoch is not.
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
		startIndex = trackedEpochsCount - window
	}

	runningVolumeForWindow := num.UintZero()
	for i := startIndex; i < trackedEpochsCount; i++ {
		runningVolumeForWindow.AddSum(runningVolumeSet[i].value)
	}

	return runningVolumeForWindow
}

func (vs *runningVolumes) RemovePriorEpoch(epoch uint64) {
	for setID, volumes := range vs.runningVolumesBySet {
		removeBeforeIndex := len(volumes) - 1
		for i := len(volumes) - 1; i >= 0; i-- {
			if volumes[i].epoch < epoch {
				break
			}
			removeBeforeIndex -= 1
		}

		if removeBeforeIndex == len(volumes)-1 {
			vs.runningVolumesBySet[setID] = []*notionalVolume{}
		} else if removeBeforeIndex >= 0 {
			vs.runningVolumesBySet[setID] = volumes[removeBeforeIndex:]
		}
	}
}

func newRunningVolumes() *runningVolumes {
	return &runningVolumes{
		maxPartyNotionalVolumeByQuantumPerEpoch: num.UintZero(),
		runningVolumesBySet:                     map[types.ReferralSetID][]*notionalVolume{},
	}
}
