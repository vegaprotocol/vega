package core_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
	uuid "github.com/satori/go.uuid"
)

func theInsurancePoolInitialBalanceForTheMarketsIs(amountstr string) error {
	amount, _ := strconv.ParseUint(amountstr, 10, 0)
	execsetup = getExecutionSetupEmptyWithInsurancePoolBalance(amount)
	return nil
}

func generalAccountForAssetBalanceIs(trader, asset, balancestr string) error {
	balance, _ := strconv.ParseUint(balancestr, 10, 0)
	acc, err := execsetup.broker.GetTraderGeneralAccount(trader, asset)
	if err != nil {
		return err
	}

	if acc.Balance != balance {
		return fmt.Errorf("invalid general account balance for asset(%s) for trader(%s), expected(%d) got(%d)",
			asset, trader, balance, acc.Balance,
		)
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

func missingTradersCancelsTheFollowingOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.GetByReference(val(row, 0), val(row, 1))
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

func tradersCancelPeggedOrdersAndClear(data *gherkin.DataTable) error {
	cancellations := make([]types.OrderCancellation, 0, len(data.Rows))
	for _, row := range data.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		mkt := val(row, 1)
		orders := execsetup.broker.GetOrdersByPartyAndMarket(trader, mkt)
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

	if len(pos) == 0 {
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
		data := execsetup.broker.GetTrades()
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

func tradersAmendsTheFollowingOrdersReference(refs *gherkin.DataTable) error {
	for _, row := range refs.Rows {
		if val(row, 0) == "trader" {
			continue
		}

		o, err := execsetup.broker.GetByReference(val(row, 0), val(row, 1))
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

		o, err := execsetup.broker.GetByReference(trader, val(row, 1))
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

func executedTrades(trades *gherkin.DataTable) error {
	var err error
	for i, row := range trades.Rows {
		if i > 0 {
			trader := val(row, 0)
			price := u64val(row, 1)
			size := u64val(row, 2)
			counterparty := val(row, 3)
			var found = false
			data := execsetup.broker.GetTrades()
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
		// | trader  | market id | side | volume | reference | offset |
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
			if o.PartyId != trader || o.Status != status || o.MarketId != id || o.Side != side || o.Size != vol || o.Price != price {
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

func clearTransferEvents() error {
	execsetup.broker.ClearTransferEvents()
	return nil
}

func clearOrderEvents() error {
	execsetup.broker.ClearOrderEvents()
	return nil
}

func clearOrdersByRef(in *gherkin.DataTable) error {
	for _, row := range in.Rows {
		trader := val(row, 0)
		if trader == "trader" {
			continue
		}
		ref := val(row, 1)
		if err := execsetup.broker.ClearOrderByReference(trader, ref); err != nil {
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
	now := execsetup.timesvc.Now.Add(time.Duration(mkt.OpeningAuction.Duration*2) * time.Second)
	execsetup.timesvc.Now = now
	// notify markets
	execsetup.timesvc.Notify(context.Background(), now)
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
