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
	Total       string
	MaxForEpoch string
	BannedUntil int64
}

type Statistic struct {
	Name        string
	Total       string
	Limit       string
	BannedUntil int64
}
