package core_test

import (
	"fmt"
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

func i32val(rows *gherkin.TableRow, idx int) int32 {
	s := rows.Cells[idx].Value
	ret, _ := strconv.ParseInt(s, 10, 0)
	return int32(ret)
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

func tifval(rows *gherkin.TableRow, idx int) (proto.Order_TimeInForce, error) {
	tif, ok := proto.Order_TimeInForce_value[rows.Cells[idx].Value]
	if !ok {
		return proto.Order_TimeInForce(tif), fmt.Errorf("invalid time in force: %v", rows.Cells[idx].Value)
	}
	return proto.Order_TimeInForce(tif), nil
}

func ordertypeval(rows *gherkin.TableRow, idx int) (proto.Order_Type, error) {
	ty, ok := proto.Order_Type_value[rows.Cells[idx].Value]
	if !ok {
		return proto.Order_Type(ty), fmt.Errorf("invalid order type: %v", rows.Cells[idx].Value)
	}
	return proto.Order_Type(ty), nil
}
