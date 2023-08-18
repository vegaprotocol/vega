Feature: Target stake

  # Market risk parameters and assets don't really matter.
  # We need to track open interest i.e. sum of all long positions across the parties and how they change over time

  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | market.stake.target.timeWindow          | 168h  |
      | market.stake.target.scalingFactor       | 1.5   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 10          | -10           | 0.1                    |
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau                    | mu | r  | sigma |
      | 0.000001      | 0.00011407711613050422 | -1 | -1 | -1    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.00025   | 0.0005             |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.1           | 1.2            | 1.4            |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | position decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | BTC        | BTC   | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | default-none     | default-eth-for-future | 2                       | 1e6                    | 1e6                       | default-futures |

    # Above, it says mark price but really I don't mind if we start
    # with an opening auction as long as at start of the scenario
    # no-one has any open positions in the market.
    # So if we want to start with an auction, trade volume 1, then close out the position.

    # T0 + 8 days so whatever open interest was there after the auction
    # this is now out of the time window.
    And time is updated to "2021-03-08T00:00:00Z"

  Scenario: Max open interest changes over time
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party | asset | amount    |
      | tt_0  | BTC   | 100000000 |
      | tt_1  | BTC   | 100000000 |
      | tt_2  | BTC   | 100000000 |
      | tt_3  | BTC   | 100000000 |

    # put some volume on the book so that others can increase their
    # positions and close out if needed too
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_0  | ETH/DEC21 | buy  | 100000 | 90    | 0                | TYPE_LIMIT | TIF_GTC | tt_0_0    |
      | tt_0  | ETH/DEC21 | sell | 100000 | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_0_1    |

    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | tt_0  | ETH/DEC21 | 2000              | 0.001 | submission |
      | lp1 | tt_0  | ETH/DEC21 | 2000              | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | tt_0  | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1          | 10     |
      | tt_0  | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1          | 10     |
 
    # nothing should have traded, we have mark price set apriori or
    # due to auction closing.
    Then the mark price should be "0" for the market "ETH/DEC21"

    # Traders 1, 2, 3 go long
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_1  | ETH/DEC21 | buy  | 1000   | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_1_0    |
      | tt_2  | ETH/DEC21 | buy  | 2000   | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |
      | tt_3  | ETH/DEC21 | buy  | 3000   | 110   | 0                | TYPE_LIMIT | TIF_GTC | tt_2_0    |

    Then the opening auction period ends for market "ETH/DEC21"

    Then the market data for the market "ETH/DEC21" should be:
      | trading mode            | auction trigger             | extension trigger           | target stake | supplied stake |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 990          | 2000           |

    # So now parties 1,2,3 are long 10+20+30 = 60.
    Then the mark price should be "110" for the market "ETH/DEC21"

    # Target stake is mark_price x max_oi x target_stake_scaling_factor x rf_short
    # rf_short should have been set above to 0.1
    # target_stake = 110 x 60 x 1.5 x 0.1
    And the target stake should be "990" for the market "ETH/DEC21"

    # T0 + 8 days + 1 hour
    When time is updated to "2021-03-08T01:00:00Z"

    # Trader 3 closes out 20
    When the parties place the following orders with ticks:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | tt_3  | ETH/DEC21 | sell | 2000   | 90    | 1                | TYPE_LIMIT | TIF_GTC | tt_2_1    |

    Then the mark price should be "90" for the market "ETH/DEC21"

    # the maximum oi over the last 7 days is still unchanged
    # target_stake = 90 x 60 x 1.5 x 0.1
    And the target stake should be "810" for the market "ETH/DEC21"

    # T0 + 15 days + 2 hour
    # so now the peak of 60 should have passed from window
    When time is updated to "2021-03-15T02:00:00Z"

    Then the mark price should be "90" for the market "ETH/DEC21"

    # target_stake = 90 x 40 x 1.5 x 0.1
    And the target stake should be "540" for the market "ETH/DEC21"
