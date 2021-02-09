package core_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"

	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

func theMarketsStartsOnAndExpiresOn(start, expires string) error {
	_, err := time.Parse("2006-01-02T15:04:05Z", start)
	if err != nil {
		return fmt.Errorf("invalid start date %v", err)
	}
	_, err = time.Parse("2006-01-02T15:04:05Z", expires)
	if err != nil {
		return fmt.Errorf("invalid expiry date %v", err)
	}
	marketStart = start
	marketExpiry = expires

	return nil
}

func theInsurancePoolInitialBalanceForTheMarketsIs(amountstr string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	execsetup = getExecutionSetupEmptyWithInsurancePoolBalance(amount)
	return nil
}

func theExecutionEngineHaveTheseMarkets(arg1 *gherkin.DataTable) error {
	mkts := []types.Market{}
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}
		mkt := baseMarket(row)
		mkts = append(mkts, mkt)
	}

	t, _ := time.Parse("2006-01-02T15:04:05Z", marketStart)
	execsetup = getExecutionTestSetup(t, mkts)

	// reset market start time and expiry for next run
	marketExpiry = defaultMarketExpiry
	marketStart = defaultMarketStart

	return nil
}

func theFollowingTraders(arg1 *gherkin.DataTable) error {
	// create the trader from the table using NotifyTraderAccount
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}

		partyID := val(row, 0)
		amount := u64val(row, 1)

		// expected general accounts for the trader
		// added expected market margin accounts
		for _, mkt := range execsetup.mkts {
			asset, err := mkt.GetAsset()
			if err != nil {
				return err
			}
			_, err = execsetup.collateral.Deposit(context.Background(), partyID, asset, amount)
			if err != nil {
				return err
			}

			if !traderHaveGeneralAccount(execsetup.accs[val(row, 0)], asset) {
				acc := account{
					Type:    types.AccountType_ACCOUNT_TYPE_GENERAL,
					Balance: u64val(row, 1),
					Asset:   asset,
				}
				execsetup.accs[val(row, 0)] = append(execsetup.accs[val(row, 0)], acc)
			}

			acc := account{
				Type:    types.AccountType_ACCOUNT_TYPE_MARGIN,
				Balance: 0,
				Market:  mkt.Id,
				Asset:   asset,
			}
			execsetup.accs[val(row, 0)] = append(execsetup.accs[val(row, 0)], acc)
		}

	}
	return nil
}

func iExpectTheTradersToHaveNewGeneralAccount(arg1 *gherkin.DataTable) error {
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}

		_, err := execsetup.broker.getTraderGeneralAccount(val(row, 0), val(row, 1))
		if err != nil {
			return fmt.Errorf("missing general account for trader=%v asset=%v", val(row, 0), val(row, 1))
		}
	}
	return nil
}

func generalAccountsBalanceIs(arg1, arg2 string) error {
	balance, _ := strconv.ParseUint(arg2, 10, 0)
	for _, mkt := range execsetup.mkts {
		asset, _ := mkt.GetAsset()
		acc, err := execsetup.broker.getTraderGeneralAccount(arg1, asset)
		if err != nil {
			return err
		}
		if acc.Balance != balance {
			return fmt.Errorf("invalid general account balance, expected %v got %v", arg2, acc.Balance)
		}
	}
	return nil
}

func haveOnlyOneAccountPerAsset(arg1 string) error {
	assets := map[string]struct{}{}

	accs := execsetup.broker.GetAccounts()
	data := make([]types.Account, 0, len(accs))
	for _, a := range accs {
		data = append(data, a.Account())
	}
	for _, acc := range data {
		if acc.Owner == arg1 && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			if _, ok := assets[acc.Asset]; ok {
				return fmt.Errorf("trader=%v have multiple account for asset=%v", arg1, acc.Asset)
			}
			assets[acc.Asset] = struct{}{}
		}
	}
	return nil
}

func haveOnlyOnMarginAccountPerMarket(arg1 string) error {
	assets := map[string]struct{}{}

	accs := execsetup.broker.GetAccounts()
	data := make([]types.Account, 0, len(accs))
	for _, a := range accs {
		data = append(data, a.Account())
	}
	for _, acc := range data {
		if acc.Owner == arg1 && acc.Type == types.AccountType_ACCOUNT_TYPE_MARGIN {
			if _, ok := assets[acc.MarketId]; ok {
				return fmt.Errorf("trader=%v have multiple account for market=%v", arg1, acc.MarketId)
			}
			assets[acc.MarketId] = struct{}{}
		}
	}
	return nil
}

func theMakesADepositOfIntoTheAccount(trader, amountstr, asset string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup

	_, err := execsetup.collateral.Deposit(
		context.Background(), trader, asset, amount,
	)
	return err
}

func generalAccountForAssetBalanceIs(trader, asset, balancestr string) error {
	balance, _ := strconv.ParseUint(balancestr, 10, 0)
	acc, err := execsetup.broker.getTraderGeneralAccount(trader, asset)
	if err != nil {
		return err
	}

	if acc.Balance != balance {
		return fmt.Errorf("invalid general asset=%v account balance=%v for trader=%v", asset, acc.Balance, trader)
	}

	return nil
}

func theWithdrawFromTheAccount(trader, amountstr, asset string) error {
	amount, err := strconv.ParseUint(amountstr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup
	if err != nil {
		return err
	}

	if _, err := execsetup.collateral.LockFundsForWithdraw(
		context.Background(), trader, asset, amount,
	); err != nil {
		return err
	}

	_, err = execsetup.collateral.Withdraw(
		context.Background(), trader, asset, amount,
	)
	return err
}

func tradersPlaceFollowingOrders(orders *gherkin.DataTable) error {
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		oty, err := ordertypeval(row, 6)
		if err != nil {
			return err
		}
		tif, err := tifval(row, 7)
		if err != nil {
			return err
		}

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			expiresAt = time.Now().Add(24 * time.Hour).UnixNano()
		}

		order := types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketId:    val(row, 1),
			PartyId:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
		}
		result, err := execsetup.engine.SubmitOrder(context.Background(), &order)
		if err != nil {
			return fmt.Errorf("unable to place order, err=%v (trader=%v)", err, val(row, 0))
		}

		if int64(len(result.Trades)) != i64val(row, 5) {
			return fmt.Errorf("expected %d trades, instead saw %d (%#v)", i64val(row, 5), len(result.Trades), *result)
		}
	}
	return nil
}

func missingTradersPlaceFollowingOrdersWithReferences(orders *gherkin.DataTable) error {
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		oty, err := ordertypeval(row, 6)
		if err != nil {
			return err
		}
		tif, err := tifval(row, 7)
		if err != nil {
			return err
		}

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			expiresAt = time.Now().Add(24 * time.Hour).UnixNano()
		}

		order := types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketId:    val(row, 1),
			PartyId:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
			Reference:   val(row, 8),
		}
		if _, err := execsetup.engine.SubmitOrder(context.Background(), &order); err == nil {
			return fmt.Errorf("expected trader %s to not exist", order.PartyId)
		}
	}
	return nil
}

func tradersPlaceFollowingOrdersWithReferences(orders *gherkin.DataTable) error {
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		oty, err := ordertypeval(row, 6)
		if err != nil {
			return err
		}
		tif, err := tifval(row, 7)
		if err != nil {
			return err
		}

		var expiresAt int64
		if oty != types.Order_TYPE_MARKET {
			expiresAt = time.Now().Add(24 * time.Hour).UnixNano()
		}

		order := types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketId:    val(row, 1),
			PartyId:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   expiresAt,
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
			Reference:   val(row, 8),
		}
		result, err := execsetup.engine.SubmitOrder(context.Background(), &order)
		if err != nil {
			return fmt.Errorf("err(%v), trader(%v), ref(%v)", err, order.PartyId, order.Reference)
		}
		if int64(len(result.Trades)) != i64val(row, 5) {
			return fmt.Errorf("expected %d trades, instead saw %d (%#v)", i64val(row, 5), len(result.Trades), *result)
		}
	}
	return nil
}

func tradersCancelsTheFollowingFilledOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.getByReference(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}

		cancel := types.OrderCancellation{
			OrderId:  o.Id,
			PartyId:  o.PartyId,
			MarketId: o.MarketId,
		}

		if _, err = execsetup.engine.CancelOrder(context.Background(), &cancel); err == nil {
			return fmt.Errorf("successfully cancelled order for trader %s (reference %s)", o.PartyId, o.Reference)
		}
	}

	return nil
}

func missingTradersCancelsTheFollowingOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.getByReference(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}

		cancel := types.OrderCancellation{
			OrderId:  o.Id,
			PartyId:  o.PartyId,
			MarketId: o.MarketId,
		}

		if _, err = execsetup.engine.CancelOrder(context.Background(), &cancel); err == nil {
			return fmt.Errorf("successfully cancelled order for trader %s (reference %s)", o.PartyId, o.Reference)
		}
	}

	return nil
}

func tradersCancelsTheFollowingOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.getFirstByReference(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}

		cancel := types.OrderCancellation{
			OrderId:  o.Id,
			PartyId:  o.PartyId,
			MarketId: o.MarketId,
		}

		_, err = execsetup.engine.CancelOrder(context.Background(), &cancel)
		if err != nil {
			return fmt.Errorf("unable to cancel order for trader %s, reference %s", o.PartyId, o.Reference)
		}
	}

	return nil
}

func tradersCancelPeggedOrders(data *gherkin.DataTable) error {
	for _, row := range data.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		cancel := types.OrderCancellation{
			PartyId:  trader,
			MarketId: val(row, 1),
			OrderId:  val(row, 2),
		}
		_, err := execsetup.engine.CancelOrder(context.Background(), &cancel)
		if err != nil {
			return fmt.Errorf("unable to cancel order: %+v", err)
		}
	}
	return nil
}

func tradersCancelPeggedOrdersAndClear(data *gherkin.DataTable) error {
	cancellations := make([]types.OrderCancellation, 0, len(data.Rows))
	for _, row := range data.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		mkt := val(row, 1)
		orders := execsetup.broker.getOrdersByPartyAndMarket(trader, mkt)
		if len(orders) == 0 {
			return fmt.Errorf("no orders found for party %s on market %s", trader, mkt)
		}
		// orders have to be pegged:
		found := false
		for _, o := range orders {
			if o.PeggedOrder != nil && o.Status != types.Order_STATUS_CANCELLED && o.Status != types.Order_STATUS_REJECTED {
				cancellations = append(cancellations, types.OrderCancellation{
					PartyId:  trader,
					MarketId: mkt,
					OrderId:  o.Id,
				})
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no valid pegged order found for %s on market %s", trader, mkt)
		}
	}
	// do the clear stuff
	if err := clearOrderEvents(); err != nil {
		return err
	}
	for _, c := range cancellations {
		if _, err := execsetup.engine.CancelOrder(context.Background(), &c); err != nil {
			return fmt.Errorf("failed to cancel pegged order %s for %s on market %s", c.OrderId, c.PartyId, c.MarketId)
		}
	}
	return nil
}

func iExpectTheTraderToHaveAMargin(arg1 *gherkin.DataTable) error {
	for _, row := range arg1.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		generalAccount, err := execsetup.broker.getTraderGeneralAccount(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}

		var hasError bool

		if generalAccount.GetBalance() != u64val(row, 4) {
			hasError = true
		}
		marginAccount, err := execsetup.broker.getTraderMarginAccount(val(row, 0), val(row, 2))
		if err != nil {
			return err
		}
		if marginAccount.GetBalance() != u64val(row, 3) {
			hasError = true
		}

		if hasError {
			return fmt.Errorf("expected balances to be margin(%d) general(%v), instead saw margin(%v), general(%v), (trader: %v)", i64val(row, 3), i64val(row, 4), marginAccount.GetBalance(), generalAccount.GetBalance(), val(row, 0))
		}

	}
	return nil
}

func allBalancesCumulatedAreWorth(amountstr string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	var cumul uint64
	batch := execsetup.broker.GetAccounts()
	data := make([]types.Account, 0, len(batch))
	for _, e := range batch {
		data = append(data, e.Account())
	}
	for _, v := range data {
		// remove vote token
		if v.Asset != "VOTE" {
			cumul += v.Balance
		}
	}

	if amount != cumul {
		return fmt.Errorf("expected cumul balances to be %v but found %v", amount, cumul)
	}
	return nil
}

func theFollowingTransfersHappened(arg1 *gherkin.DataTable) error {
	for _, row := range arg1.Rows {
		from := val(row, 0)
		if from == "from" {
			continue
		}

		fromAccountID := accountID(val(row, 4), from, val(row, 6), types.AccountType_value[val(row, 2)])
		toAccountID := accountID(val(row, 4), val(row, 1), val(row, 6), types.AccountType_value[val(row, 3)])

		var ledgerEntry *types.LedgerEntry
		transferEvents := execsetup.broker.GetTransferResponses()
		data := make([]*types.TransferResponse, 0, len(transferEvents))
		for _, e := range transferEvents {
			data = append(data, e.TransferResponses()...)
		}

		for _, v := range data {
			for _, _v := range v.GetTransfers() {
				if _v.FromAccount == fromAccountID && _v.ToAccount == toAccountID {
					if _v.Amount != u64val(row, 5) {
						continue
					}
					ledgerEntry = _v
					break
				}
			}
			if ledgerEntry != nil {
				break
			}
		}

		if ledgerEntry == nil {
			return fmt.Errorf("missing transfers between %v and %v for amount %v", fromAccountID, toAccountID, i64val(row, 5))
		}
		if ledgerEntry.Amount != u64val(row, 5) {
			return fmt.Errorf("invalid amount transfer %v and %v", ledgerEntry.Amount, i64val(row, 5))
		}
	}

	execsetup.broker.ResetType(events.TransferResponses)
	return nil
}

func theSettlementAccountBalanceIsForTheMarketBeforeMTM(amountstr, market string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	acc, err := execsetup.broker.getMarketSettlementAccount(market)
	if err != nil {
		return err
	}
	if amount != acc.Balance {
		return fmt.Errorf("invalid balance for market settlement account, expected %v, got %v", amount, acc.Balance)
	}
	return nil
}

func theInsurancePoolBalanceIsForTheMarket(amountstr, market string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	acc, err := execsetup.broker.getMarketInsurancePoolAccount(market)
	if err != nil {
		return err
	}
	if amount != acc.Balance {
		return fmt.Errorf("invalid balance for market insurance pool, expected %v, got %v", amount, acc.Balance)
	}
	return nil
}

func theTimeIsUpdatedTo(newTime string) error {
	t, err := time.Parse("2006-01-02T15:04:05Z", newTime)
	if err != nil {
		return fmt.Errorf("invalid start date %v", err)
	}

	execsetup.timesvc.SetTime(t)
	return nil
}

func tradersCannotPlaceTheFollowingOrdersAnymore(orders *gherkin.DataTable) error {
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		order := types.Order{
			Id:          uuid.NewV4().String(),
			MarketId:    val(row, 1),
			PartyId:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        types.Order_TYPE_LIMIT,
			TimeInForce: types.Order_TIME_IN_FORCE_GTT,
			CreatedAt:   time.Now().UnixNano(),
		}
		_, err := execsetup.engine.SubmitOrder(context.Background(), &order)
		if err == nil {
			return fmt.Errorf("expected error (%v) but got (%v)", val(row, 6), err)
		}
		if err.Error() != val(row, 6) {
			return fmt.Errorf("expected error (%v) but got (%v)", val(row, 6), err)
		}
	}
	return nil
}

func theMarginsLevelsForTheTradersAre(traders *gherkin.DataTable) error {
	for _, row := range traders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		partyID, marketID := val(row, 0), val(row, 1)
		ml, err := execsetup.broker.getMarginByPartyAndMarket(partyID, marketID)
		if err != nil {
			return err
		}

		var hasError bool

		if ml.MaintenanceMargin != u64val(row, 2) {
			hasError = true
		}
		if ml.SearchLevel != u64val(row, 3) {
			hasError = true
		}
		if ml.InitialMargin != u64val(row, 4) {
			hasError = true
		}
		if ml.CollateralReleaseLevel != u64val(row, 5) {
			hasError = true
		}
		if hasError {
			return fmt.Errorf(
				"invalid margins, expected maintenance(%v), search(%v), initial(%v), release(%v) but got maintenance(%v), search(%v), initial(%v), release(%v) (trader=%v)", i64val(row, 2), i64val(row, 3), i64val(row, 4), i64val(row, 5), ml.MaintenanceMargin, ml.SearchLevel, ml.InitialMargin, ml.CollateralReleaseLevel, val(row, 0))
		}
	}
	return nil
}

func tradersPlaceFollowingFailingOrders(orders *gherkin.DataTable) error {
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		oty, err := ordertypeval(row, 6)
		if err != nil {
			return err
		}

		tif := types.Order_TIME_IN_FORCE_GTT
		if len(row.Cells) > 7 {
			tif, err = tifval(row, 7)
			if err != nil {
				return err
			}
		}

		order := types.Order{
			Id:          uuid.NewV4().String(),
			MarketId:    val(row, 1),
			PartyId:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
		}
		_, err = execsetup.engine.SubmitOrder(context.Background(), &order)
		if err == nil {
			return fmt.Errorf("expected error (%v) but got (%v)", val(row, 5), err)
		}
		if err.Error() != val(row, 5) {
			return fmt.Errorf("expected error (%v) but got (%v)", val(row, 5), err)
		}
	}
	return nil
}

func theFollowingOrdersAreRejected(orders *gherkin.DataTable) error {
	ordCnt := len(orders.Rows) - 1
	for _, row := range orders.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		data := execsetup.broker.GetOrderEvents()
		for _, o := range data {
			v := o.Order()
			if v.PartyId == val(row, 0) && v.MarketId == val(row, 1) &&
				v.Status == types.Order_STATUS_REJECTED && v.Reason.String() == val(row, 2) {
				ordCnt -= 1
			}
		}
	}

	if ordCnt > 0 {
		return errors.New("some orders were not rejected")
	}
	return nil
}

func positionAPIProduceTheFollowingRow(row *gherkin.TableRow) (err error) {
	var retries = 2

	party, volume, realisedPNL, unrealisedPNL := val(row, 0), i64val(row, 1), i64val(row, 3), i64val(row, 2)

	var pos []*types.Position
	sleepTime := 100 // milliseconds
	for retries > 0 {
		pos, err = execsetup.positionPlugin.GetPositionsByParty(party)
		if err != nil {
			// Do not retry. Fail immediately.
			return fmt.Errorf("error getting party position, party(%v), err(%v)", party, err)
		}

		if len(pos) == 1 && pos[0].OpenVolume == volume && pos[0].RealisedPnl == realisedPNL && pos[0].UnrealisedPnl == unrealisedPNL {
			return nil
		}

		// The positions engine runs asynchronously, so wait for the right numbers to show up.
		// Sleep times: 100ms, 200ms, 400ms, ..., 51.2s, then give up.
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		sleepTime *= 2
		retries--
	}

	if pos == nil || len(pos) <= 0 {
		return fmt.Errorf("party do not have a position, party(%v)", party)
	}

	return fmt.Errorf("invalid positions api values for party(%v): volume (expected %v, got %v), unrealisedPNL (expected %v, got %v), realisedPNL (expected %v, got %v)",
		party, volume, pos[0].OpenVolume, unrealisedPNL, pos[0].UnrealisedPnl, realisedPNL, pos[0].RealisedPnl)
}

func positionAPIProduceTheFollowing(table *gherkin.DataTable) error {
	for _, row := range table.Rows {
		if val(row, 0) == "trader" {
			continue
		}
		if err := positionAPIProduceTheFollowingRow(row); err != nil {
			return err
		}
	}
	return nil
}

func theMarkPriceForTheMarketIs(market, markPriceStr string) error {
	markPrice, err := strconv.ParseUint(markPriceStr, 10, 0)
	if err != nil {
		return fmt.Errorf("markPrice is not a integer: markPrice(%v), err(%v)", markPriceStr, err)
	}

	mktdata, err := execsetup.engine.GetMarketData(market)
	if err != nil {
		return fmt.Errorf("unable to get mark price for market(%v), err(%v)", markPriceStr, err)
	}

	if mktdata.MarkPrice != markPrice {
		return fmt.Errorf("mark price if wrong for market(%v), expected(%v) got(%v)", market, markPrice, mktdata.MarkPrice)
	}

	return nil
}

func theMarketTradingModeIs(market, marketTradingModeStr string) error {
	ms, ok := types.Market_TradingMode_value[marketTradingModeStr]
	if !ok {
		return fmt.Errorf("invalid market state: %v", marketTradingModeStr)
	}
	marketTradingMode := types.Market_TradingMode(ms)

	mktdata, err := execsetup.engine.GetMarketData(market)
	if err != nil {
		return fmt.Errorf("unable to get marked data for market(%v), err(%v)", market, err)
	}

	if mktdata.MarketTradingMode != marketTradingMode {
		return fmt.Errorf("market trading mode is wrong for market(%v), expected(%v) got(%v)", market, marketTradingMode, mktdata.MarketTradingMode)
	}
	return nil
}

func theFollowingNetworkTradesHappened(trades *gherkin.DataTable) error {
	var err error
	for _, row := range trades.Rows {
		if val(row, 0) == "trader" {
			continue
		}
		ok := false
		party, side, volume := val(row, 0), sideval(row, 1), u64val(row, 2)
		data := execsetup.broker.getTrades()
		for _, v := range data {
			if (v.Buyer == party || v.Seller == party) && v.Aggressor == side && v.Size == volume {
				ok = true
				break
			}
		}

		if !ok {
			err = fmt.Errorf("expecting trade was missing: %v, %v, %v", party, side, volume)
			break
		}
	}

	return err
}

func theFollowingTradesHappened(trades *gherkin.DataTable) error {
	var err error
	for _, row := range trades.Rows {
		if val(row, 0) == "buyer" {
			continue
		}
		buyer, seller, price, volume := val(row, 0), val(row, 1), u64val(row, 2), u64val(row, 3)
		data := execsetup.broker.getTrades()
		for _, v := range data {
			if v.Buyer == buyer && v.Seller == seller && v.Price == price && v.Size == volume {
				return nil
			}
		}

		return fmt.Errorf("expecting trade was missing: buyer(%v), seller(%v), price(%v), volume(%v)", buyer, seller, price, volume)
	}

	return err
}

func tradersAmendsTheFollowingOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.getByReference(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}

		tif, err := tifval(row, 5)
		if err != nil {
			return fmt.Errorf("invalid time in for ref(%v)", val(row, 5))
		}

		success, err := boolval(row, 6)
		if err != nil {
			return err
		}

		value := u64val(row, 2)
		var price *types.Price
		if value != 0 {
			price = &types.Price{Value: value}
		}

		amend := types.OrderAmendment{
			OrderId:     o.Id,
			PartyId:     o.PartyId,
			MarketId:    o.MarketId,
			Price:       price,
			SizeDelta:   i64val(row, 3),
			TimeInForce: tif,
		}

		_, err = execsetup.engine.AmendOrder(context.Background(), &amend)
		if err != nil && success {
			return fmt.Errorf("expected to succeed amending but failed for trader %s (reference %s, err %v)", o.PartyId, o.Reference, err)
		}

		if err == nil && !success {
			return fmt.Errorf("expected to failed amending but succeed for trader %s (reference %s)", o.PartyId, o.Reference)
		}

	}

	return nil
}

func verifyTheStatusOfTheOrderReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}

		o, err := execsetup.broker.getByReference(trader, val(row, 1))
		if err != nil {
			return err
		}

		status, err := orderstatusval(row, 2)
		if err != nil {
			return err
		}
		if status != o.Status {
			return fmt.Errorf("invalid order status for order ref %v, expected %v got %v", o.Reference, status.String(), o.Status.String())
		}
	}

	return nil
}

func dumpTransfers() error {
	fmt.Println("DUMPING TRANSFERS")
	transferEvents := execsetup.broker.GetTransferResponses()
	for _, e := range transferEvents {
		for _, t := range e.TransferResponses() {
			for _, v := range t.GetTransfers() {
				fmt.Printf("transfer: %v\n", *v)
			}
		}
	}
	return nil
}

func accountID(marketID, partyID, asset string, _ty int32) string {
	ty := types.AccountType(_ty)
	idbuf := make([]byte, 256)
	if ty == types.AccountType_ACCOUNT_TYPE_GENERAL {
		marketID = ""
	}
	if partyID == "market" {
		partyID = ""
	}
	const (
		systemOwner = "*"
		noMarket    = "!"
	)
	if len(marketID) <= 0 {
		marketID = noMarket
	}

	// market account
	if len(partyID) <= 0 {
		partyID = systemOwner
	}

	copy(idbuf, marketID)
	ln := len(marketID)
	copy(idbuf[ln:], partyID)
	ln += len(partyID)
	copy(idbuf[ln:], asset)
	ln += len(asset)
	idbuf[ln] = byte(ty + 48)
	return string(idbuf[:ln+1])
}

func baseMarket(row *gherkin.TableRow) types.Market {
	horizons, err := i64arr(row, 21, ",")
	if err != nil {
		log.Fatalf("Can't parse horizons (%v) to int64 array: %v", row.Cells[21].Value, err)
	}
	probs, err := f64arr(row, 22, ",")
	if err != nil {
		log.Fatalf("Can't parse probabilities (%v) to float64 array: %v", row.Cells[22].Value, err)
	}
	durations, err := i64arr(row, 23, ",")
	if err != nil {
		log.Fatalf("Can't parse durations (%v) to int64 array: %v", row.Cells[23].Value, err)
	}
	n := len(horizons)
	if n != len(probs) || n != len(durations) {
		log.Fatalf("horizons (%v), probabilities (%v) and durations (%v) need to have the same number of elements",
			n,
			len(probs),
			len(durations))
	}

	triggs := make([]*types.PriceMonitoringTrigger, 0, n)
	for i := 0; i < n; i++ {
		p := &types.PriceMonitoringTrigger{Horizon: horizons[i], Probability: probs[i], AuctionExtension: durations[i]}
		triggs = append(triggs, p)
	}
	pMonitorSettings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: triggs,
		},
		UpdateFrequency: i64val(row, 20),
	}

	openingAuction := &types.AuctionDuration{
		Duration: i64val(row, 15),
	}

	if openingAuction.Duration <= 0 {
		openingAuction = nil
	}

	oracleSpecPropertyType, err := oracleSpecPropertyTypeVal(row, 27)
	if err != nil {
		log.Fatalf("Can't parse oracleSpecPropertyType %v to PropertyKey_Type: %v", row.Cells[28].Value, err)
	}

	mkt := types.Market{
		TradingMode:   types.Market_TRADING_MODE_CONTINUOUS,
		State:         types.Market_STATE_ACTIVE,
		Id:            val(row, 0),
		DecimalPlaces: 2,
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				LiquidityFee:      val(row, 19),
				InfrastructureFee: val(row, 18),
				MakerFee:          val(row, 17),
			},
		},
		TradableInstrument: &types.TradableInstrument{
			Instrument: &types.Instrument{
				Id:   fmt.Sprintf("Crypto/%s/Futures", val(row, 0)),
				Code: fmt.Sprintf("CRYPTO/%v", val(row, 0)),
				Name: fmt.Sprintf("%s future", val(row, 0)),
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				InitialMarkPrice: u64val(row, 4),
				Product: &types.Instrument_Future{
					Future: &types.Future{
						Maturity: marketExpiry,
						SettlementAsset: val(row, 3),
						QuoteName:       val(row, 2),
						OracleSpec: &oraclesv1.OracleSpecConfiguration{
							PubKeys: strings.Split(val(row, 25),","),
							Filters: []*oraclesv1.Filter{
								{
									Key: &oraclesv1.PropertyKey{
										Name: val(row, 26),
										Type: oracleSpecPropertyType,
									},
									Conditions: []*oraclesv1.Condition{},
								},
							},
						},
						OracleSpecBinding: &types.OracleSpecToFutureBinding{
							SettlementPriceProperty: val(row, 28),
						},
					},
				},
			},
			RiskModel: &types.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &types.SimpleRiskModel{
					Params: &types.SimpleModelParams{
						FactorLong:           f64val(row, 6),
						FactorShort:          f64val(row, 7),
						MaxMoveUp:            f64val(row, 8),
						MinMoveDown:          f64val(row, 9),
						ProbabilityOfTrading: f64val(row, 24),
					},
				},
			},
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       f64val(row, 13),
					InitialMargin:     f64val(row, 12),
					CollateralRelease: f64val(row, 11),
				},
			},
		},
		OpeningAuction: openingAuction,
		TradingModeConfig: &types.Market_Continuous{
			Continuous: &types.ContinuousTrading{},
		},
		PriceMonitoringSettings: pMonitorSettings,
		TargetStakeParameters: &types.TargetStakeParameters{
			TimeWindow:    3600,
			ScalingFactor: 10,
		},
	}
	if val(row, 5) == "forward" {
		mkt.TradableInstrument.RiskModel = &types.TradableInstrument_LogNormalRiskModel{
			LogNormalRiskModel: &types.LogNormalRiskModel{
				RiskAversionParameter: f64val(row, 6), // 6
				Tau:                   f64val(row, 7), // 7
				Params: &types.LogNormalModelParams{
					Mu:    f64val(row, 8),  // 8
					R:     f64val(row, 9),  // 9
					Sigma: f64val(row, 10), //10
				},
			},
		}
	}

	return mkt
}

func executedTrades(trades *gherkin.DataTable) error {
	var err error
	for i, row := range trades.Rows {
		if i > 0 {
			trader := val(row, 0)
			price := u64val(row, 1)
			size := u64val(row, 2)
			counterparty := val(row, 3)
			var found = false
			data := execsetup.broker.getTrades()
			for _, v := range data {
				if v.Buyer == trader && v.Seller == counterparty && v.Price == price && v.Size == size {
					found = true
					break
				}
			}

			if !found {
				err = fmt.Errorf("expected trade is missing: %v, %v, %v, %v", trader, price, size, counterparty)
				break
			}
		}
	}

	return err
}

func dumpOrders() error {
	fmt.Println("DUMPING ORDERS")
	data := execsetup.broker.GetOrderEvents()
	for _, v := range data {
		o := *v.Order()
		fmt.Printf("order %s: %v\n", o.Id, o)
	}
	return nil
}

func dumpTrades() error {
	fmt.Println("DUMPING TRADES")
	data := execsetup.broker.getTrades()
	for _, t := range data {
		fmt.Printf("trade %s, %#v\n", t.Id, t)
	}
	return nil
}

func tradersPlacePeggedOrders(orders *gherkin.DataTable) error {
	for i, row := range orders.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		id, side, vol, ref, offset, price := val(row, 1), val(row, 2), u64val(row, 3), peggedRef(row, 4), i64val(row, 5), u64val(row, 6)
		o := &types.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Type:        types.Order_TYPE_LIMIT,
			TimeInForce: types.Order_TIME_IN_FORCE_GTC,
			Id:          "someid",
			Side:        types.Side_SIDE_BUY,
			PartyId:     trader,
			MarketId:    id,
			Size:        vol,
			Price:       price,
			Remaining:   vol,
			Reference:   fmt.Sprintf("%s-pegged-order-%d", trader, i),
			PeggedOrder: &types.PeggedOrder{
				Reference: ref,
				Offset:    offset,
			},
		}
		if side == "sell" {
			o.Side = types.Side_SIDE_SELL
		}
		_, err := execsetup.engine.SubmitOrder(context.Background(), o)
		if err != nil {
			fmt.Println("DUMP ORDER ERROR")
			fmt.Printf("Error: %v\n", err)
			fmt.Println("DUMP ORDER")
			fmt.Printf("%#v\n", *o)
			return err
		}
	}
	return nil
}

func seeTheFollowingOrderEvents(evts *gherkin.DataTable) error {
	data := execsetup.broker.GetOrderEvents()
	for _, row := range evts.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		// | trader   | id        | side | volume | reference | offset |
		id, sside, vol, ref, offset, price := val(row, 1),
			val(row, 2), u64val(row, 3), peggedRef(row, 4), i64val(row, 5), u64val(row, 6)
		status, err := orderstatusval(row, 7)
		if err != nil {
			return err
		}
		side := types.Side_SIDE_BUY
		if sside == "sell" {
			side = types.Side_SIDE_SELL
		}
		match := false
		for _, e := range data {
			o := e.Order()
			if o.Status != status || o.MarketId != id || o.Side != side || o.Size != vol || o.Price != price {
				// if o.MarketId != id || o.Side != side || o.Size != vol || o.Price != price {
				continue
			}
			// check if pegged:
			if offset != 0 {
				// nope
				if o.PeggedOrder == nil {
					continue
				}
				if o.PeggedOrder.Offset != offset || o.PeggedOrder.Reference != ref {
					continue
				}
				// this matches
			}
			// we've checked all fields and found this order to be a match
			match = true
			break
		}
		if !match {
			return errors.New("no matching order event found")
		}
	}
	return nil
}

func clearOrderEvents() error {
	execsetup.broker.clearOrderEvents()
	return nil
}

func clearOrdersByRef(in *gherkin.DataTable) error {
	for _, row := range in.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		ref := val(row, 1)
		if err := execsetup.broker.clearOrderByReference(trader, ref); err != nil {
			return err
		}
	}
	return nil
}

// liquidity provisioning
func submitLP(in *gherkin.DataTable) error {
	lps := map[string]*types.LiquidityProvisionSubmission{}
	parties := map[string]string{}
	// build the LPs to submit
	for _, row := range in.Rows {
		id := val(row, 0)
		if id == "id" {
			continue
		}
		lp, ok := lps[id]
		if !ok {
			lp = &types.LiquidityProvisionSubmission{
				MarketId:         val(row, 2),
				CommitmentAmount: u64val(row, 3),
				Fee:              val(row, 4),
				Sells:            []*types.LiquidityOrder{},
				Buys:             []*types.LiquidityOrder{},
			}
			parties[id] = val(row, 1)
			lps[id] = lp
		}
		lo := &types.LiquidityOrder{
			Reference:  peggedRef(row, 6),
			Proportion: uint32(u64val(row, 7)),
			Offset:     i64val(row, 8),
		}
		if side := val(row, 5); side == "buy" {
			lp.Buys = append(lp.Buys, lo)
		} else {
			lp.Sells = append(lp.Sells, lo)
		}
	}
	for id, sub := range lps {
		party, ok := parties[id]
		if !ok {
			return errors.New("party for LP not found")
		}
		if err := execsetup.engine.SubmitLiquidityProvision(context.Background(), sub, party, id); err != nil {
			return err
		}
	}
	return nil
}

func seeLPEvents(in *gherkin.DataTable) error {
	evts := execsetup.broker.GetLPEvents()
	evtByID := func(id string) *types.LiquidityProvision {
		for _, e := range evts {
			if lp := e.LiquidityProvision(); lp.Id == id {
				return &lp
			}
		}
		return nil
	}
	for _, row := range in.Rows {
		id := val(row, 0)
		if id == "id" {
			continue
		}
		// find event
		e := evtByID(id)
		if e == nil {
			return errors.New("no LP for id found")
		}
		party, market, commitment := val(row, 1), val(row, 2), u64val(row, 3)
		if e.PartyId != party || e.MarketId != market || e.CommitmentAmount != commitment {
			return errors.New("party,  market ID, or commitment amount mismatch")
		}
	}
	return nil
}

func theOpeningAuctionPeriodEnds(mktName string) error {
	var mkt *types.Market
	for _, m := range execsetup.mkts {
		if m.Id == mktName {
			mkt = &m
			break
		}
	}
	if mkt == nil {
		return fmt.Errorf("market %s not found", mktName)
	}
	// double the time, so it's definitely past opening auction time
	now := execsetup.timesvc.now.Add(time.Duration(mkt.OpeningAuction.Duration*2) * time.Second)
	execsetup.timesvc.now = now
	// notify markets
	execsetup.timesvc.notify(context.Background(), now)
	return nil
}

func tradersWithdrawBalance(in *gherkin.DataTable) error {
	for _, row := range in.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		asset, amount := val(row, 1), u64val(row, 2)
		if _, err := execsetup.collateral.LockFundsForWithdraw(context.Background(), trader, asset, amount); err != nil {
			return err
		}
		if _, err := execsetup.collateral.Withdraw(context.Background(), trader, asset, amount); err != nil {
			return err
		}
	}
	return nil
}

func oracleSendsFutureSettlementPriceEventAt(price, pubKey string) error {
	return execsetup.oracleEngine.BroadcastData(context.Background(), oracles.OracleData{
		PubKeys: []string{pubKey},
		Data:    map[string]string {
			"prices.ETH.value": price,
		},
	})
}
