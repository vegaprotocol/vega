package entities

type PagedEntity interface {
	Market | Party | Trade
	Cursor() *Cursor
}

func PageEntities[T PagedEntity](items []T, pagination Pagination) ([]T, PageInfo) {
	var pagedItems []T
	var limit int
	var pageInfo PageInfo

	if len(items) == 0 {
		return pagedItems, pageInfo
	}

	if pagination.HasForward() && pagination.Forward.Limit != nil {
		limit = int(*pagination.Forward.Limit)
		switch len(items) {
		case limit + 2:
			pagedItems = items[1 : limit+1]
			pageInfo.HasNextPage = true
			pageInfo.HasPreviousPage = true
		case limit + 1:
			if !pagination.Forward.HasCursor() {
				pagedItems = items[0:limit]
				pageInfo.HasNextPage = true
				pageInfo.HasPreviousPage = false
			} else {
				pagedItems = items[1:]
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = true
			}
		default:
			// if the pagination for the first item is the same as the after pagination, then we have a previous page, and we shouldn't include it
			if pagination.HasForward() && pagination.Forward.HasCursor() && pagination.Forward.Cursor.Value() == items[0].Cursor().Value() {
				pagedItems = items[1:]
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = true
			} else {
				pagedItems = items
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = false
			}
		}
	} else if pagination.HasBackward() && pagination.Backward.Limit != nil {
		limit = int(*pagination.Backward.Limit)
		switch len(items) {
		case limit + 2:
			pagedItems = reverseSlice(items[1 : limit+1])
			pageInfo.HasNextPage = true
			pageInfo.HasPreviousPage = true
		case limit + 1:
			if !pagination.Backward.HasCursor() {
				pagedItems = reverseSlice(items[0:limit])
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = true
			} else {
				pagedItems = reverseSlice(items[1:])
				pageInfo.HasNextPage = true
				pageInfo.HasPreviousPage = false
			}
		default:
			pagedItems = reverseSlice(items)
			if pagination.HasBackward() && pagination.Backward.HasCursor() && pagination.Backward.Cursor.Value() == pagedItems[0].Cursor().Value() {
				pagedItems = pagedItems[1:]
				pageInfo.HasNextPage = true
				pageInfo.HasPreviousPage = false
			} else {
				pageInfo.HasNextPage = false
				pageInfo.HasPreviousPage = false
			}
		}
	} else {
		pagedItems = items
		pageInfo.HasNextPage = false
		pageInfo.HasPreviousPage = false
	}

	if len(pagedItems) > 0 {
		pageInfo.StartCursor = pagedItems[0].Cursor().Encode()
		pageInfo.EndCursor = pagedItems[len(pagedItems)-1].Cursor().Encode()
	}

	return pagedItems, pageInfo
}

func reverseSlice[T any](input []T) (reversed []T) {
	reversed = make([]T, len(input))
	copy(reversed, input)
	for i, j := 0, len(input)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = input[j], input[i]
	}
	return
}
