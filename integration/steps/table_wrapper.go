package steps

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	oraclesv1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
)

// StrictParseFirstRow parses and verifies, table integrity and returns only the
// first row. This is suitable of table that act more as object than actual
// table.
func StrictParseFirstRow(table *godog.Table, required, optional []string) RowWrapper {
	rows := StrictParseTable(table, required, optional)

	if len(rows) > 1 {
		panic("this table supports only one row")
	}

	return rows[0]
}

// StrictParseTable parses and verifies the table integrity.
func StrictParseTable(dt *godog.Table, required, optional []string) []RowWrapper {
	tableLen := len(dt.Rows)
	if tableLen < 1 {
		panic("A table is required.")
	}

	if len(required) != 0 {
		err := verifyTableIntegrity(required, optional, dt.Rows[0])
		if err != nil {
			panic(err)
		}
	}

	tableWithoutHeaderLen := tableLen - 1
	if tableWithoutHeaderLen == 0 {
		panic("Did you forget the table header?")
	}

	out := make([]RowWrapper, 0, tableWithoutHeaderLen)
	for _, row := range dt.Rows[1:] {
		wrapper := RowWrapper{values: map[string]string{}}
		for i := range row.Cells {
			wrapper.values[dt.Rows[0].Cells[i].Value] = row.Cells[i].Value
		}
		out = append(out, wrapper)
	}

	return out
}

// ParseTable parses the table without verifying its integrity.
// Prefer the use of StrictParseTable().
func ParseTable(dt *godog.Table) []RowWrapper {
	return StrictParseTable(dt, []string{}, []string{})
}

func verifyTableIntegrity(required, optional []string, header *messages.PickleStepArgument_PickleTable_PickleTableRow) error {
	cols, err := newColumns(required, optional)
	if err != nil {
		return err
	}

	var headerNames []string
	for _, cell := range header.Cells {
		headerNames = append(headerNames, cell.Value)
	}

	return cols.Verify(headerNames)
}

type columns struct {
	// config maps a column name to it required state.
	// true == required
	// false == optional
	config map[string]bool
}

func newColumns(required []string, optional []string) (*columns, error) {
	config := map[string]bool{}

	for _, column := range required {
		config[column] = true
	}

	for _, optColumn := range optional {
		_, ok := config[optColumn]
		if ok {
			return nil, fmt.Errorf("column \"%s\" can't be required and optional at the same time", optColumn)
		}
		config[optColumn] = false
	}

	return &columns{
		config: config,
	}, nil
}

// Verify ensures the table declares the expected columns and does
// not declared any unexpected columns.
func (c *columns) Verify(header []string) error {
	declaredColumnsSet := map[string]interface{}{}

	for _, column := range header {
		_, ok := c.config[column]
		if !ok {
			return fmt.Errorf("the column \"%s\" is not expected by this table", column)
		}
		declaredColumnsSet[column] = nil
	}

	for column, isRequired := range c.config {
		_, ok := declaredColumnsSet[column]
		if !ok && isRequired {
			return fmt.Errorf("the column \"%s\" is required by this table", column)
		}
	}

	return nil
}

type RowWrapper struct {
	values map[string]string
}

func (r RowWrapper) mustColumn(name string) string {
	s, ok := r.values[name]
	if !ok {
		panic(fmt.Errorf("column \"%s\" not found", name))
	}
	return s
}

func (r RowWrapper) HasColumn(name string) bool {
	if v, ok := r.values[name]; !ok || v == "" {
		return false
	}
	return true
}

func (r RowWrapper) MustStr(name string) string {
	return r.mustColumn(name)
}

// StrB does the same as Str, but returns a bool indicating whether or not the
// column was set
func (r RowWrapper) StrB(name string) (string, bool) {
	return r.Str(name), r.HasColumn(name)
}

func (r RowWrapper) Str(name string) string {
	return r.values[name]
}

func (r RowWrapper) MustStrSlice(name, sep string) []string {
	return StrSlice(r.mustColumn(name), sep)
}

func (r RowWrapper) StrSlice(name, sep string) []string {
	return StrSlice(r.values[name], sep)
}

func StrSlice(value string, sep string) []string {
	if len(value) == 0 {
		return nil
	}
	return strings.Split(value, sep)
}

func (r RowWrapper) MustDecimal(name string) num.Decimal {
	value, err := Decimal(r.mustColumn(name))
	panicW(name, err)
	return value
}

func (r RowWrapper) Decimal(name string) num.Decimal {
	value, err := Decimal(r.values[name])
	panicW(name, err)
	return value
}

func Decimal(rawValue string) (num.Decimal, error) {
	return num.DecimalFromString(rawValue)
}

func (r RowWrapper) MustU64(name string) uint64 {
	value, err := U64(r.mustColumn(name))
	panicW(name, err)
	return value
}

func (r RowWrapper) MustUint(name string) *num.Uint {
	value, err := Uint(r.mustColumn(name))
	panicW(name, err)
	return value
}

func (r RowWrapper) Uint(name string) *num.Uint {
	value, err := Uint(r.values[name])
	panicW(name, err)
	return value
}

func Uint(value string) (*num.Uint, error) {
	retVal, overflow := num.UintFromString(value, 10)
	if overflow {
		return nil, fmt.Errorf("invalid uint value: %v", value)
	}
	return retVal, nil
}

// U64B does the same as U64, but returns a bool indicating whether or not the
// column was set
func (r RowWrapper) U64B(name string) (uint64, bool) {
	if !r.HasColumn(name) {
		return 0, false
	}
	return r.U64(name), true
}

func (r RowWrapper) U64(name string) uint64 {
	value, err := U64(r.values[name])
	panicW(name, err)
	return value
}

func U64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 0)
}

func (r RowWrapper) MustU32(name string) uint32 {
	value, err := U64(r.mustColumn(name))
	panicW(name, err)
	return uint32(value)
}

func (r RowWrapper) U32(name string) uint32 {
	value, err := U64(r.values[name])
	panicW(name, err)
	return uint32(value)
}

func (r RowWrapper) MustU64Slice(name, sep string) []uint64 {
	value, err := U64Slice(r.mustColumn(name), sep)
	panicW(name, err)
	return value
}

func (r RowWrapper) U64Slice(name, sep string) []uint64 {
	value, err := U64Slice(r.values[name], sep)
	panicW(name, err)
	return value
}

func U64Slice(rawValue, sep string) ([]uint64, error) {
	if len(rawValue) == 0 {
		return []uint64{}, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]uint64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseUint(rawValues[i], 10, 0)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) MustI64(name string) int64 {
	value, err := I64(r.mustColumn(name))
	panicW(name, err)
	return value
}

// I64B does the same as U64B, but returns a bool indicating whether or not the
// column was set
func (r RowWrapper) I64B(name string) (int64, bool) {
	if !r.HasColumn(name) {
		return 0, false
	}
	return r.I64(name), true
}

func (r RowWrapper) I64(name string) int64 {
	value, err := I64(r.values[name])
	panicW(name, err)
	return value
}

func I64(rawValue string) (int64, error) {
	return strconv.ParseInt(rawValue, 10, 0)
}

func (r RowWrapper) MustI64Slice(name, sep string) []int64 {
	value, err := I64Slice(r.mustColumn(name), sep)
	panicW(name, err)
	return value
}

func (r RowWrapper) I64Slice(name, sep string) []int64 {
	value, err := I64Slice(r.values[name], sep)
	panicW(name, err)
	return value
}

func I64Slice(rawValue string, sep string) ([]int64, error) {
	if len(rawValue) == 0 {
		return []int64{}, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]int64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseInt(rawValues[i], 10, 0)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) MustF64(name string) float64 {
	value, err := F64(r.mustColumn(name))
	panicW(name, err)
	return value
}

func (r RowWrapper) F64(name string) float64 {
	value, err := F64(r.values[name])
	panicW(name, err)
	return value
}

func F64(rawValue string) (float64, error) {
	return strconv.ParseFloat(rawValue, 64)
}

func (r RowWrapper) MustF64Slice(name, sep string) []float64 {
	value, err := F64Slice(r.mustColumn(name), sep)
	panicW(name, err)
	return value
}

func (r RowWrapper) F64Slice(name, sep string) []float64 {
	value, err := F64Slice(r.values[name], sep)
	panicW(name, err)
	return value
}

func F64Slice(rawValue string, sep string) ([]float64, error) {
	if len(rawValue) == 0 {
		return nil, nil
	}
	rawValues := strings.Split(rawValue, sep)
	valuesCount := len(rawValues)
	array := make([]float64, 0, valuesCount)
	for i := 0; i < valuesCount; i++ {
		item, err := strconv.ParseFloat(rawValues[i], 64)
		if err != nil {
			return nil, err
		}
		array = append(array, item)
	}
	return array, nil
}

func (r RowWrapper) MustBool(name string) bool {
	b, err := Bool(r.mustColumn(name))
	panicW(name, err)
	return b
}

func (r RowWrapper) Bool(name string) bool {
	b, err := Bool(r.values[name])
	panicW(name, err)
	return b
}

func Bool(rawValue string) (bool, error) {
	if rawValue == "true" {
		return true, nil
	} else if rawValue == "false" {
		return false, nil
	}
	return false, fmt.Errorf("invalid bool value: %v", rawValue)
}

func (r RowWrapper) MustTime(name string) time.Time {
	t, err := Time(r.mustColumn(name))
	panicW(name, err)
	return t
}

func (r RowWrapper) Time(name string) time.Time {
	t, err := Time(r.values[name])
	panicW(name, err)
	return t
}

func Time(rawTime string) (time.Time, error) {
	parsedTime, err := time.Parse("2006-01-02T15:04:05Z", rawTime)
	if err != nil {
		return parsedTime, fmt.Errorf("invalid date value: %v", err)
	}
	return parsedTime, nil
}

func (r RowWrapper) MustOrderType(name string) types.OrderType {
	orderType, err := OrderType(r.MustStr(name))
	panicW(name, err)
	return orderType
}

func OrderType(rawValue string) (types.OrderType, error) {
	ty, ok := proto.Order_Type_value[rawValue]
	if !ok {
		return types.OrderType(ty), fmt.Errorf("invalid order type: %v", rawValue)
	}
	return types.OrderType(ty), nil
}

func (r RowWrapper) MustOrderStatus(name string) types.OrderStatus {
	s, err := OrderStatus(r.MustStr(name))
	panicW(name, err)
	return s
}

func OrderStatus(rawValue string) (types.OrderStatus, error) {
	ty, ok := proto.Order_Status_value[rawValue]
	if !ok {
		return types.OrderStatus(ty), fmt.Errorf("invalid order status: %v", rawValue)
	}
	return types.OrderStatus(ty), nil
}

func (r RowWrapper) MustLiquidityStatus(name string) types.LiquidityProvisionStatus {
	s, err := LiquidityStatus(r.MustStr(name))
	panicW(name, err)
	return s
}

func LiquidityStatus(rawValue string) (types.LiquidityProvisionStatus, error) {
	ty, ok := proto.LiquidityProvision_Status_value[rawValue]
	if !ok {
		return types.LiquidityProvisionStatus(ty), fmt.Errorf("invalid liquidity provision status: %v", rawValue)
	}
	return types.LiquidityProvisionStatus(ty), nil
}

func (r RowWrapper) MustTIF(name string) types.OrderTimeInForce {
	tif, err := TIF(r.MustStr(name))
	panicW(name, err)
	return tif
}

func TIF(rawValue string) (types.OrderTimeInForce, error) {
	tif, ok := proto.Order_TimeInForce_value[strings.ReplaceAll(rawValue, "TIF_", "TIME_IN_FORCE_")]
	if !ok {
		return types.OrderTimeInForce(tif), fmt.Errorf("invalid time in force: %v", rawValue)
	}
	return types.OrderTimeInForce(tif), nil
}

func (r RowWrapper) MustSide(name string) types.Side {
	side, err := Side(r.MustStr(name))
	panicW(name, err)
	return side
}

func Side(rawValue string) (types.Side, error) {
	switch rawValue {
	case "sell":
		return types.SideSell, nil
	case "buy":
		return types.SideBuy, nil
	default:
		return types.SideUnspecified, errors.New("invalid side")
	}
}

func (r RowWrapper) MustPeggedReference(name string) types.PeggedReference {
	return peggedReference(r.MustStr(name))
}

func peggedReference(rawValue string) types.PeggedReference {
	switch rawValue {
	case "MID":
		return types.PeggedReferenceMid
	case "ASK":
		return types.PeggedReferenceBestAsk
	case "BID":
		return types.PeggedReferenceBestBid
	}
	return types.PeggedReferenceUnspecified
}

func (r RowWrapper) MustOracleSpecPropertyType(name string) oraclesv1.PropertyKey_Type {
	ty, err := OracleSpecPropertyType(r.MustStr(name))
	panicW(name, err)
	return ty
}

func OracleSpecPropertyType(name string) (oraclesv1.PropertyKey_Type, error) {
	ty, ok := oraclesv1.PropertyKey_Type_value[name]

	if !ok {
		return oraclesv1.PropertyKey_TYPE_UNSPECIFIED, fmt.Errorf("couldn't find %s as property type", name)
	}
	return oraclesv1.PropertyKey_Type(ty), nil
}

func (r RowWrapper) MustAuctionTrigger(name string) types.AuctionTrigger {
	at, err := AuctionTrigger(r.MustStr(name))
	panicW(name, err)
	return at
}

func AuctionTrigger(name string) (types.AuctionTrigger, error) {
	at, ok := proto.AuctionTrigger_value[name]
	if !ok {
		return types.AuctionTriggerUnspecified, fmt.Errorf("couldn't find %s as auction trigger", name)
	}
	return types.AuctionTrigger(at), nil
}

func (r RowWrapper) MustTradingMode(name string) types.MarketTradingMode {
	ty, err := TradingMode(r.MustStr(name))
	panicW(name, err)
	return ty
}

func TradingMode(name string) (types.MarketTradingMode, error) {
	ty, ok := proto.Market_TradingMode_value[name]

	if !ok {
		return types.MarketTradingModeUnspecified, fmt.Errorf("couldn't find %s as trading_mode", name)
	}
	return types.MarketTradingMode(ty), nil
}

func MarketState(name string) (types.MarketState, error) {
	ty, ok := proto.Market_State_value[name]

	if !ok {
		return types.MarketStateUnspecified, fmt.Errorf("couldn't find %s as market state", name)
	}
	return types.MarketState(ty), nil
}

func (r RowWrapper) MustAccount(name string) types.AccountType {
	acc, err := Account(r.MustStr(name))
	panicW(name, err)
	return acc
}

func Account(name string) (types.AccountType, error) {
	value := types.AccountType(proto.AccountType_value[name])

	if value == types.AccountTypeUnspecified {
		return types.AccountTypeUnspecified, fmt.Errorf("invalid account type %s", name)
	}
	return value, nil
}

func AccountID(marketID, partyID, asset string, ty types.AccountType) string {
	idBuf := make([]byte, 256)

	if ty == types.AccountTypeGeneral || ty == types.AccountTypeFeesInfrastructure {
		marketID = ""
	}

	if partyID == "market" {
		partyID = ""
	}

	if len(marketID) == 0 {
		marketID = "!"
	}

	if len(partyID) == 0 {
		partyID = "*"
	}

	copy(idBuf, marketID)
	ln := len(marketID)
	copy(idBuf[ln:], partyID)
	ln += len(partyID)
	copy(idBuf[ln:], asset)
	ln += len(asset)
	idBuf[ln] = byte(ty + 48)
	return string(idBuf[:ln+1])
}

func (r RowWrapper) MustPrice(name string) *types.Price {
	n := r.MustUint(name)
	// nil instead of zero value of Price is expected by APIs
	if n.IsZero() {
		return nil
	}
	return Price(n)
}

func Price(n *num.Uint) *types.Price {
	return &types.Price{Value: n}
}

func (r RowWrapper) MustDuration(name string) time.Duration {
	return time.Duration(r.MustU64(name))
}

func (r RowWrapper) Duration(name string) time.Duration {
	return time.Duration(r.U64(name))
}

func (r RowWrapper) MustDurationStr(name string) time.Duration {
	s := r.MustStr(name)
	d, err := time.ParseDuration(s)
	panicW(name, err)
	return d
}

func (r RowWrapper) MustDurationSec(name string) time.Duration {
	n := r.MustU64(name)
	if n == 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func (r RowWrapper) DurationSec(name string) time.Duration {
	n := r.U64(name)
	if n == 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func panicW(field string, err error) {
	if err != nil {
		panic(fmt.Sprintf("couldn't parse %s: %v", field, err))
	}
}

func stringToU64(s string) uint64 {
	i, _ := strconv.ParseUint(s, 10, 64)
	return i
}
