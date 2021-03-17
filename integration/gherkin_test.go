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

func i64val(rows *gherkin.TableRow, idx int) int64 {
	s := rows.Cells[idx].Value
	ret, _ := strconv.ParseInt(s, 10, 0)
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

func orderstatusval(rows *gherkin.TableRow, idx int) (proto.Order_Status, error) {
	st, ok := proto.Order_Status_value[rows.Cells[idx].Value]
	if !ok {
		return proto.Order_Status(st), fmt.Errorf("invalid time in force: %v", rows.Cells[idx].Value)
	}
	return proto.Order_Status(st), nil
}

func ordertypeval(rows *gherkin.TableRow, idx int) (proto.Order_Type, error) {
	ty, ok := proto.Order_Type_value[rows.Cells[idx].Value]
	if !ok {
		return proto.Order_Type(ty), fmt.Errorf("invalid order type: %v", rows.Cells[idx].Value)
	}
	return proto.Order_Type(ty), nil
}

func boolval(rows *gherkin.TableRow, idx int) (bool, error) {
	val := rows.Cells[idx].Value
	if val == "true" {
		return true, nil
	} else if val == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value: %v", val)
}

func peggedRef(rows *gherkin.TableRow, i int) proto.PeggedReference {
	switch rows.Cells[i].Value {
	case "MID":
		return proto.PeggedReference_PEGGED_REFERENCE_MID
	case "ASK":
		return proto.PeggedReference_PEGGED_REFERENCE_BEST_ASK
	case "BID":
		return proto.PeggedReference_PEGGED_REFERENCE_BEST_BID
	}
	return proto.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED
}
