package spam

type Statistics struct {
	Proposals         Statistic
	Delegations       Statistic
	Transfers         Statistic
	NodeAnnouncements Statistic
	Votes             VoteStatistic
}

type VoteStatistic struct {
	Total        string
	BlockedUntil int64
}

type Statistic struct {
	Total        string
	Limit        string
	BlockedUntil int64
}
