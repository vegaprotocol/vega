package core_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/proto"
	types "code.vegaprotocol.io/vega/proto"

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

func theExecutonEngineHaveTheseMarkets(arg1 *gherkin.DataTable) error {
	mkts := []proto.Market{}
	for _, row := range arg1.Rows {
		if val(row, 0) == "name" {
			continue
		}
		mkt := baseMarket(row)
		mkts = append(mkts, mkt)
	}

	t, _ := time.Parse("2006-01-02T15:04:05Z", marketStart)
	execsetup = getExecutionTestSetup(t, mkts)

	// reset market startime and expiry for next run
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

		// row.0 = traderID, row.1 = amount to topup
		notif := proto.NotifyTraderAccount{
			TraderID: val(row, 0),
			Amount:   u64val(row, 1),
		}

		err := execsetup.engine.NotifyTraderAccount(context.Background(), &notif)
		if err != nil {
			return err
		}

		// expected general accounts for the trader
		// added expected market margin accounts
		for _, mkt := range execsetup.mkts {
			asset, err := mkt.GetAsset()
			if err != nil {
				return err
			}

			if !traderHaveGeneralAccount(execsetup.accs[val(row, 0)], asset) {
				acc := account{
					Type:    proto.AccountType_ACCOUNT_TYPE_GENERAL,
					Balance: u64val(row, 1),
					Asset:   asset,
				}
				execsetup.accs[val(row, 0)] = append(execsetup.accs[val(row, 0)], acc)
			}

			acc := account{
				Type:    proto.AccountType_ACCOUNT_TYPE_MARGIN,
				Balance: 0,
				Market:  mkt.Name,
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
		if uint64(acc.Balance) != balance {
			return fmt.Errorf("invalid general account balance, expected %v got %v", arg2, acc.Balance)
		}
	}
	return nil
}

func haveOnlyOneAccountPerAsset(arg1 string) error {
	assets := map[string]struct{}{}

	accs := execsetup.broker.GetAccounts()
	data := make([]proto.Account, 0, len(accs))
	for _, a := range accs {
		data = append(data, a.Account())
	}
	for _, acc := range data {
		if acc.Owner == arg1 && acc.Type == proto.AccountType_ACCOUNT_TYPE_GENERAL {
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
	data := make([]proto.Account, 0, len(accs))
	for _, a := range accs {
		data = append(data, a.Account())
	}
	for _, acc := range data {
		if acc.Owner == arg1 && acc.Type == proto.AccountType_ACCOUNT_TYPE_MARGIN {
			if _, ok := assets[acc.MarketID]; ok {
				return fmt.Errorf("trader=%v have multiple account for market=%v", arg1, acc.MarketID)
			}
			assets[acc.MarketID] = struct{}{}
		}
	}
	return nil
}

func theMakesADepositOfIntoTheAccount(trader, amountstr, asset string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup
	notif := proto.NotifyTraderAccount{
		TraderID: trader,
		Amount:   amount,
	}

	err := execsetup.engine.NotifyTraderAccount(context.Background(), &notif)
	if err != nil {
		return err
	}

	return nil
}

func generalAccountForAssetBalanceIs(trader, asset, balancestr string) error {
	balance, _ := strconv.ParseUint(balancestr, 10, 0)
	acc, err := execsetup.broker.getTraderGeneralAccount(trader, asset)
	if err != nil {
		return err
	}

	if uint64(acc.Balance) != balance {
		return fmt.Errorf("invalid general asset=%v account balance=%v for trader=%v", asset, acc.Balance, trader)
	}

	return nil
}

func theWithdrawFromTheAccount(trader, amountstr, asset string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	// row.0 = traderID, row.1 = amount to topup
	notif := proto.Withdraw{
		PartyID: trader,
		Amount:  amount,
		Asset:   asset,
	}

	err := execsetup.engine.Withdraw(context.Background(), &notif)
	if err != nil {
		return err
	}

	return nil

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

		order := proto.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketID:    val(row, 1),
			PartyID:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
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

		order := proto.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketID:    val(row, 1),
			PartyID:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
			Reference:   val(row, 8),
		}
		if _, err := execsetup.engine.SubmitOrder(context.Background(), &order); err == nil {
			return fmt.Errorf("expected trader %s to not exist", order.PartyID)
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

		order := proto.Order{
			Status:      types.Order_STATUS_ACTIVE,
			Id:          uuid.NewV4().String(),
			MarketID:    val(row, 1),
			PartyID:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        oty,
			TimeInForce: tif,
			CreatedAt:   time.Now().UnixNano(),
			Reference:   val(row, 8),
		}
		result, err := execsetup.engine.SubmitOrder(context.Background(), &order)
		if err != nil {
			return err
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

		cancel := proto.OrderCancellation{
			OrderID:  o.Id,
			PartyID:  o.PartyID,
			MarketID: o.MarketID,
		}

		if _, err = execsetup.engine.CancelOrder(context.Background(), &cancel); err == nil {
			return fmt.Errorf("successfully cancelled order for trader %s (reference %s)", o.PartyID, o.Reference)
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

		cancel := proto.OrderCancellation{
			OrderID:  o.Id,
			PartyID:  o.PartyID,
			MarketID: o.MarketID,
		}

		if _, err = execsetup.engine.CancelOrder(context.Background(), &cancel); err == nil {
			return fmt.Errorf("successfully cancelled order for trader %s (reference %s)", o.PartyID, o.Reference)
		}
	}

	return nil
}

func tradersCancelsTheFollowingOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.getByReference(val(row, 0), val(row, 1))
		if err != nil {
			return err
		}

		cancel := proto.OrderCancellation{
			OrderID:  o.Id,
			PartyID:  o.PartyID,
			MarketID: o.MarketID,
		}

		_, err = execsetup.engine.CancelOrder(context.Background(), &cancel)
		if err != nil {
			return fmt.Errorf("unable to cancel order for trader %s, reference %s", o.PartyID, o.Reference)
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
		if v.Asset != collateral.TokenAsset {
			cumul += uint64(v.Balance)
		}
	}

	if amount != cumul {
		return fmt.Errorf("expected cumul balances to be %v but found %v", amount, cumul)
	}
	return nil
}

func theFollowingTransfersHappend(arg1 *gherkin.DataTable) error {
	for _, row := range arg1.Rows {
		if val(row, 0) == "from" {
			continue
		}

		fromAccountID := accountID(val(row, 4), val(row, 0), val(row, 6), proto.AccountType_value[val(row, 2)])
		toAccountID := accountID(val(row, 4), val(row, 1), val(row, 6), proto.AccountType_value[val(row, 3)])

		var ledgerEntry *proto.LedgerEntry
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

		order := proto.Order{
			Id:          uuid.NewV4().String(),
			MarketID:    val(row, 1),
			PartyID:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        proto.Order_TYPE_LIMIT,
			TimeInForce: proto.Order_TIF_GTT,
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
		ml, err := execsetup.marginLevelsBuf.getMarginByPartyAndMarket(partyID, marketID)
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

		order := proto.Order{
			Id:          uuid.NewV4().String(),
			MarketID:    val(row, 1),
			PartyID:     val(row, 0),
			Side:        sideval(row, 2),
			Price:       u64val(row, 4),
			Size:        u64val(row, 3),
			Remaining:   u64val(row, 3),
			ExpiresAt:   time.Now().Add(24 * time.Hour).UnixNano(),
			Type:        oty,
			TimeInForce: proto.Order_TIF_GTT,
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
			if v.PartyID == val(row, 0) && v.MarketID == val(row, 1) &&
				v.Status == proto.Order_STATUS_REJECTED && v.Reason.String() == val(row, 2) {
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

	var pos []*proto.Position
	sleepTime := 100 // milliseconds
	for retries > 0 {
		pos, err = execsetup.positionPlugin.GetPositionsByParty(party)
		if err != nil {
			// Do not retry. Fail immediately.
			return fmt.Errorf("error getting party position, party(%v), err(%v)", party, err)
		}

		if len(pos) == 1 && pos[0].OpenVolume == volume && pos[0].RealisedPNL == realisedPNL && pos[0].UnrealisedPNL == unrealisedPNL {
			return nil
		}

		// The positions engine runs asynchronously, so wait for the right numbers to show up.
		// Sleep times: 100ms, 200ms, 400ms, ..., 51.2s, then give up.
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		sleepTime *= 2
		retries--
	}

	if len(pos) <= 0 {
		return fmt.Errorf("party do not have a position, party(%v)", party)
	}

	return fmt.Errorf("invalid positions api values for party(%v): volume (expected %v, got %v), unrealisedPNL (expected %v, got %v), realisedPNL (expected %v, got %v)",
		party, volume, pos[0].OpenVolume, unrealisedPNL, pos[0].UnrealisedPNL, realisedPNL, pos[0].RealisedPNL)
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
		ok := false
		buyer, seller, price, volume := val(row, 0), val(row, 1), u64val(row, 2), u64val(row, 3)
		data := execsetup.broker.getTrades()
		for _, v := range data {
			if v.Buyer == buyer && v.Seller == seller && v.Price == price && v.Size == volume {
				ok = true
				break
			}
		}

		if !ok {
			err = fmt.Errorf("expecting trade was missing: buyer(%v), seller(%v), price(%v), volume(%v)", buyer, seller, price, volume)
			break
		}
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
		var price *proto.Price
		if value != 0 {
			price = &proto.Price{Value: value}
		}

		amend := proto.OrderAmendment{
			OrderID:     o.Id,
			PartyID:     o.PartyID,
			MarketID:    o.MarketID,
			Price:       price,
			SizeDelta:   i64val(row, 3),
			TimeInForce: tif,
		}

		_, err = execsetup.engine.AmendOrder(context.Background(), &amend)
		if err != nil && success {
			return fmt.Errorf("expected to succeed amending but failed for trader %s (reference %s, err %v)", o.PartyID, o.Reference, err)
		}

		if err == nil && !success {
			return fmt.Errorf("expected to failed amending but succeed for trader %s (reference %s)", o.PartyID, o.Reference)
		}

	}

	return nil
}

func verifyTheStatusOfTheOrderReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.getByReference(val(row, 0), val(row, 1))
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
	ty := proto.AccountType(_ty)
	idbuf := make([]byte, 256)
	if ty == proto.AccountType_ACCOUNT_TYPE_GENERAL {
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

func baseMarket(row *gherkin.TableRow) proto.Market {
	mkt := proto.Market{
		Id:            val(row, 0),
		Name:          val(row, 0),
		DecimalPlaces: 2,
		TradableInstrument: &proto.TradableInstrument{
			Instrument: &proto.Instrument{
				Id:        fmt.Sprintf("Crypto/%s/Futures", val(row, 0)),
				Code:      fmt.Sprintf("CRYPTO/%v", val(row, 0)),
				Name:      fmt.Sprintf("%s future", val(row, 0)),
				BaseName:  val(row, 1),
				QuoteName: val(row, 2),
				Metadata: &proto.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				InitialMarkPrice: u64val(row, 4),
				Product: &proto.Instrument_Future{
					Future: &proto.Future{
						Maturity: marketExpiry,
						Oracle: &proto.Future_EthereumEvent{
							EthereumEvent: &proto.EthereumEvent{
								ContractID: "0x0B484706fdAF3A4F24b2266446B1cb6d648E3cC1",
								Event:      "price_changed",
								Value:      u64val(row, 14),
							},
						},
						Asset: val(row, 3),
					},
				},
			},
			RiskModel: &proto.TradableInstrument_SimpleRiskModel{
				SimpleRiskModel: &proto.SimpleRiskModel{
					Params: &proto.SimpleModelParams{
						FactorLong:  f64val(row, 6),
						FactorShort: f64val(row, 7),
					},
				},
			},
			MarginCalculator: &proto.MarginCalculator{
				ScalingFactors: &proto.ScalingFactors{
					SearchLevel:       f64val(row, 13),
					InitialMargin:     f64val(row, 12),
					CollateralRelease: f64val(row, 11),
				},
			},
		},
		TradingMode: &proto.Market_Continuous{
			Continuous: &proto.ContinuousTrading{},
		},
	}

	if val(row, 5) == "forward" {
		mkt.TradableInstrument.RiskModel = &proto.TradableInstrument_LogNormalRiskModel{
			LogNormalRiskModel: &proto.LogNormalRiskModel{
				RiskAversionParameter: f64val(row, 6), // 6
				Tau:                   f64val(row, 7), // 7
				Params: &proto.LogNormalModelParams{
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
			var found bool = false
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
	data := execsetup.broker.GetOrderEvents()
	for _, v := range data {
		o := *v.Order()
		fmt.Printf("order %s: %v\n", o.Id, o)
	}
	return nil
}
