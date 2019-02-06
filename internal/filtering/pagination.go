package filtering

type QueryFilterPaginated struct {
	First *uint64
	Last  *uint64
	Skip  *uint64
}

func (q *QueryFilterPaginated) HasFirst() bool {
	return q.First != nil && *q.First > uint64(0)
}
func (q *QueryFilterPaginated) HasLast() bool {
	return q.Last != nil && *q.Last > uint64(0)
}
func (q *QueryFilterPaginated) HasSkip() bool {
	return q.Skip != nil && *q.Skip > uint64(0)
}

