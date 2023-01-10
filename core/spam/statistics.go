package spam

type Statistics struct {
	Proposals         Statistic
	Delegations       Statistic
	Transfers         Statistic
	NodeAnnouncements Statistic
	Votes             []VoteStatistic
}

type VoteStatistic struct {
	Proposal    string
	Total       uint64
	MaxForEpoch uint64
	BannedUntil int64
}

type Statistic struct {
	Name        string
	Total       uint64
	Limit       uint64
	BannedUntil int64
}
