package core_test

import (
	"strconv"

	"code.vegaprotocol.io/vega/proto"
	"github.com/DATA-DOG/godog/gherkin"
)

func val(rows *gherkin.TableRow, idx int) string {
	return rows.Cells[idx].Value
}

func u64val(rows *gherkin.TableRow, idx int) uint64 {
	s := rows.Cells[idx].Value
	ret, _ := strconv.ParseUint(s, 10, 0)
	return ret
}

func i64val(rows *gherkin.TableRow, idx int) int64 {
	s := rows.Cells[idx].Value
	ret, _ := strconv.ParseInt(s, 10, 0)
	return ret
}

func f64val(rows *gherkin.TableRow, idx int) float64 {
	s := rows.Cells[idx].Value
	ret, _ := strconv.ParseFloat(s, 10)
	return ret
}

func sideval(rows *gherkin.TableRow, idx int) proto.Side {
	s := rows.Cells[idx].Value
	if s == "sell" {
		return proto.Side_Sell
	}
	return proto.Side_Buy
}
