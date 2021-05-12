package steps

import "strconv"

func TheBlockTimeIs(bt string) (int64, error) {
	return strconv.ParseInt(bt, 10, 0)
}
