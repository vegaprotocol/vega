package core_test

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/proto"
)

func val(rows *gherkin.TableRow, idx int) string {
	return rows.Cells[idx].Value
}

func u64val(rows *gherkin.TableRow, idx int) uint64 {
	s := rows.Cells[idx].Value
	ret, _ := strconv.ParseUint(s, 10, 0)
	return ret
}

func sideval(rows *gherkin.TableRow, idx int) proto.Side {
	s := rows.Cells[idx].Value
	if s == "sell" {
		return proto.Side_SIDE_SELL
	}
	return proto.Side_SIDE_BUY
}

func tifval(rows *gherkin.TableRow, idx int) (proto.Order_TimeInForce, error) {
	tif, ok := proto.Order_TimeInForce_value[strings.ReplaceAll(rows.Cells[idx].Value, "TIF_", "TIME_IN_FORCE_")]
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
