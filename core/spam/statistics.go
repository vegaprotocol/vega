package spam

type Statistics struct {
	Proposals         Statistic
	Delegations       Statistic
	Transfers         Statistic
	NodeAnnouncements Statistic
	Votes             VoteStatistic
}

type VoteStatistic struct {
	Total         string
	Rejected      string
	RejectedRatio string
	Limit         string
	BlockedUntil  int64
}

type Statistic struct {
	Total        string
	BlockCount   string
	Limit        string
	BlockedUntil int64
}
