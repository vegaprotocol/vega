package fee

import (
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"golang.org/x/exp/maps"
)

type FeeStats struct {
	TotalRewardsPaid        map[string]*num.Uint
	ReferrerRewardsGenerate map[string]map[string]*num.Uint
	RefereeDiscountApplied  map[string]*num.Uint
	VolumeDiscountApplied   map[string]*num.Uint
}

func NewFeeStats() *FeeStats {
	return &FeeStats{
		TotalRewardsPaid:        map[string]*num.Uint{},
		ReferrerRewardsGenerate: map[string]map[string]*num.Uint{},
		RefereeDiscountApplied:  map[string]*num.Uint{},
		VolumeDiscountApplied:   map[string]*num.Uint{},
	}
}

func NewFeeStatsFromProto(fsp *eventspb.FeeStats) *FeeStats {
	fs := NewFeeStats()

	for _, v := range fsp.RefereesDiscountApplied {
		fs.RefereeDiscountApplied[v.Party] = num.MustUintFromString(v.Amount, 10)
	}

	for _, v := range fsp.VolumeDiscountApplied {
		fs.VolumeDiscountApplied[v.Party] = num.MustUintFromString(v.Amount, 10)
	}

	for _, v := range fsp.TotalRewardsPaid {
		fs.TotalRewardsPaid[v.Party] = num.MustUintFromString(v.Amount, 10)
	}

	for _, v := range fsp.ReferrerRewardsGenerated {
		rg := map[string]*num.Uint{}
		for _, pa := range v.GeneratedReward {
			rg[pa.Party] = num.MustUintFromString(pa.Amount, 10)
		}

		fs.ReferrerRewardsGenerate[v.Referrer] = rg
	}

	return fs
}

func (f *FeeStats) RegisterReferrerReward(
	referrer, referee string,
	amount *num.Uint,
) {
	total, ok := f.TotalRewardsPaid[referrer]
	if !ok {
		total = num.NewUint(0)
		f.TotalRewardsPaid[referrer] = total
	}

	total.Add(total, amount)

	rewardsGenerated, ok := f.ReferrerRewardsGenerate[referrer]
	if !ok {
		rewardsGenerated = map[string]*num.Uint{}
		f.ReferrerRewardsGenerate[referrer] = rewardsGenerated
	}

	refereeTally, ok := rewardsGenerated[referee]
	if !ok {
		refereeTally = num.NewUint(0)
		rewardsGenerated[referrer] = refereeTally
	}

	refereeTally.Add(refereeTally, amount)
}

func (f *FeeStats) RegisterRefereeDiscount(party string, amount *num.Uint) {
	total, ok := f.RefereeDiscountApplied[party]
	if !ok {
		total = num.NewUint(0)
		f.RefereeDiscountApplied[party] = total
	}

	total.Add(total, amount)
}

func (f *FeeStats) RegisterVolumeDiscount(party string, amount *num.Uint) {
	total, ok := f.VolumeDiscountApplied[party]
	if !ok {
		total = num.NewUint(0)
		f.VolumeDiscountApplied[party] = total
	}

	total.Add(total, amount)
}

func (f *FeeStats) ToProto(asset string) *eventspb.FeeStats {
	fs := &eventspb.FeeStats{
		Asset:                    asset,
		TotalRewardsPaid:         make([]*eventspb.PartyAmount, 0, len(f.TotalRewardsPaid)),
		ReferrerRewardsGenerated: make([]*eventspb.ReferrerRewardsGenerated, 0, len(f.ReferrerRewardsGenerate)),
		RefereesDiscountApplied:  make([]*eventspb.PartyAmount, 0, len(f.RefereeDiscountApplied)),
		VolumeDiscountApplied:    make([]*eventspb.PartyAmount, 0, len(f.VolumeDiscountApplied)),
	}

	totalRewardsPaidParties := maps.Keys(f.TotalRewardsPaid)
	sort.Strings(totalRewardsPaidParties)
	for _, party := range totalRewardsPaidParties {
		amount := f.TotalRewardsPaid[party]
		fs.TotalRewardsPaid = append(fs.TotalRewardsPaid, &eventspb.PartyAmount{
			Party:  party,
			Amount: amount.String(),
		})
	}

	refereesDiscountAppliedParties := maps.Keys(f.RefereeDiscountApplied)
	sort.Strings(refereesDiscountAppliedParties)
	for _, party := range refereesDiscountAppliedParties {
		amount := f.RefereeDiscountApplied[party]
		fs.RefereesDiscountApplied = append(fs.RefereesDiscountApplied, &eventspb.PartyAmount{
			Party:  party,
			Amount: amount.String(),
		})
	}

	volumeDiscountAppliedParties := maps.Keys(f.RefereeDiscountApplied)
	sort.Strings(volumeDiscountAppliedParties)
	for _, party := range volumeDiscountAppliedParties {
		amount := f.RefereeDiscountApplied[party]
		fs.VolumeDiscountApplied = append(fs.VolumeDiscountApplied, &eventspb.PartyAmount{
			Party:  party,
			Amount: amount.String(),
		})
	}

	referrerRewardsGeneratedParties := maps.Keys(f.ReferrerRewardsGenerate)
	sort.Strings(referrerRewardsGeneratedParties)
	for _, party := range referrerRewardsGeneratedParties {
		partiesAmounts := f.ReferrerRewardsGenerate[party]

		rewardsGenerated := &eventspb.ReferrerRewardsGenerated{
			Referrer:        party,
			GeneratedReward: make([]*eventspb.PartyAmount, 0, len(partiesAmounts)),
		}

		partiesAmountsParties := maps.Keys(partiesAmounts)
		sort.Strings(partiesAmountsParties)
		for _, party := range partiesAmountsParties {
			amount := partiesAmounts[party]
			rewardsGenerated.GeneratedReward = append(
				rewardsGenerated.GeneratedReward,
				&eventspb.PartyAmount{
					Party:  party,
					Amount: amount.String(),
				},
			)
		}

		fs.ReferrerRewardsGenerated = append(fs.ReferrerRewardsGenerated, rewardsGenerated)
	}

	return fs
}
