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
