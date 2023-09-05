Feature: Test liquidity provider reward distribution; Should also cover liquidity-fee-setting and equity-like-share calc and total stake.
  # to look into and test: If an equity-like share is small and LP rewards are distributed immediately, then how do we round? (does a small share get rounded up or down, do they all add up?)
  #Check what happens with time and distribution period (both in genesis and mid-market)

  Background:

    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 8     |
      | validators.epoch.length                             | 24h   |
      | market.liquidity.providersFeeCalculationTimeStep  | 600s  |
    
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.5                    | 0                         | SLA        |

    Given the average block duration is "2"

  Scenario: 001: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades (0042-LIQF-001)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 4         | 1                    | buy  | BID              | 4          | 2      |
      | lp1   | ETH/MAR22 | 7         | 1                    | buy  | MID              | 7          | 1      |
      | lp1   | ETH/MAR22 | 4         | 1                    | sell | ASK              | 4          | 2      |
      | lp1   | ETH/MAR22 | 7         | 1                    | sell | MID              | 7          | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 1000         | 10000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    # max_oi: max open interest

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 4      |
      | buy  | 900   | 1      |
      | buy  | 999   | 7      |
      | sell | 1001  | 7      |
      | sell | 1100  | 1      |
      | sell | 1102  | 4      |

    #volume = ceiling(liquidity_obligation x liquidity-normalised-proportion / price)
    #priceLvel at 898: 10000 * (1/3) / 898 = 4
    #priceLvel at 999: 10000 * (2/3) / 999 = 7
    #priceLvel at 1001: 10000 * (2/3) / 1001 = 7
    #priceLvel at 1102: 10000 * (1/3) / 1102 = 4

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 10000                   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 1320   | 999988680 | 10000 |
      | party1 | USD   | ETH/MAR22 | 1704   | 99998296  |       |
      | party2 | USD   | ETH/MAR22 | 1692   | 99998308  |       |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 1         | 1                    | sell | MID              | 1          | 1      |
      | lp1   | ETH/MAR22 | 1         | 1                    | buy  | MID              | 1          | 1      |


    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | lp1    | -8     | -392           | 0            |
      | party1 | -2     | 0              | 0            |
      | party2 | 10     | 0              | 392          |

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 8         | 1                    | sell | MID              | 8          | 1      |
      | lp1   | ETH/MAR22 | 1         | 1                    | buy  | MID              | 1          | 1      |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 2400   | 999987212 | 10000 |
      | party1 | USD   | ETH/MAR22 | 1200   | 99998805  |       |
      | party2 | USD   | ETH/MAR22 | 1932   | 99998411  |       |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 20     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    Then time is updated to "2019-11-30T00:20:05Z"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    # here we get only a trade for a volume of 8 as it's what was on the LP
    # order, then the 32 remaining from party1 are cancelled for self trade
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 8    | lp1    |

    # this is slightly different than expected, as the trades happen against the LP,
    # which is probably not what you expected initially
    And the accumulated liquidity fees should be "8" for the market "ETH/MAR22"

    Then time is updated to "2019-11-30T00:30:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 8      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    # Move beyond the end of the current epoch to trigger distribution to general account 
    When time is updated to "2019-12-04T00:30:05Z"
    And the network moves ahead "1" blocks
    Then the following transfers should happen:
      | from  | to  | from account                   | to account           | market id | amount | asset |
      | lp1   | lp1 | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 28     | USD   |

  Scenario: 002: 2 LPs joining at start, equal commitments (0042-LIQF-002)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp1   | ETH/MAR22 | 4         | 1                    | sell | MID              | 4          | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | ETH/MAR22 | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp2   | ETH/MAR22 | 4         | 1                    | sell | MID              | 4          | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"
    And the open interest should be "90" for the market "ETH/MAR22"
    And the target stake should be "9000" for the market "ETH/MAR22"
    And the supplied stake should be "10000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 5000                    |
      | lp2   | 0.5               | 10000                   |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.002" for the market "ETH/MAR22"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 4    | lp1    |
      | party2 | 951   | 4    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "40" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 20     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 20     | USD   |
    When the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp1   | ETH/MAR22 | 4         | 1                    | sell | MID              | 4          | 1      |
      | lp2   | ETH/MAR22 | 4         | 1                    | buy  | MID              | 4          | 1      |
      | lp2   | ETH/MAR22 | 4         | 1                    | sell | MID              | 4          | 1      |
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 4    | lp1    |
      | party1 | 951   | 4    | lp2    |
    And the accumulated liquidity fees should be "16" for the market "ETH/MAR22"

    When time is updated to "2019-11-30T00:20:08Z"
    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 8      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 8      | USD   |

  Scenario: 003: 2 LPs joining at start, unequal commitments (0042-LIQF-003)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | submission |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 2          | 1      |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 2          | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | submission |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 2          | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | MID              | 2          | 1      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"
    And the open interest should be "60" for the market "ETH/MAR22"
    And the target stake should be "6000" for the market "ETH/MAR22"
    And the supplied stake should be "10000" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 8000                    |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 0.8 vs 0.2 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 16     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 4      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 6          | 1      |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | MID              | 6          | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | MID              | 2          | 1      |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | MID              | 2          | 1      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 6    | lp1    |
      | party1 | 951   | 2    | lp2    |

    And the accumulated liquidity fees should be "8" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 0.8 vs 0.2 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 6      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 1      | USD   |

    And the accumulated liquidity fees should be "1" for the market "ETH/MAR22"

  @FeeRound
  Scenario: 004 2 LPs joining at start, unequal commitments, lp2's equity like share is very small, check the rounding of equity like share (round to 16 decimal places in this case)(0042-LIQF-004, 0042-LIQF-015)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | lp3    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 8000000           | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2970      | 1                    | buy  | BID              | 2970   | 2      |
      | lp1   | ETH/MAR22 | 5339      | 1                    | buy  | MID              | 5339   | 1      |
      | lp1   | ETH/MAR22 | 2420      | 1                    | sell | ASK              | 2420   | 2      |
      | lp1   | ETH/MAR22 | 5329      | 1                    | sell | MID              | 5329   | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 1                 | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | ETH/MAR22 | 1         | 1                    | buy  | BID              | 1          | 2      |
      | lp2   | ETH/MAR22 | 1         | 1                    | buy  | MID              | 1          | 1      |
      | lp2   | ETH/MAR22 | 1         | 1                    | sell | ASK              | 1          | 2      |
      | lp2   | ETH/MAR22 | 1         | 1                    | sell | MID              | 1          | 1      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"
    And the open interest should be "60" for the market "ETH/MAR22"
    And the target stake should be "6000" for the market "ETH/MAR22"
    And the supplied stake should be "8000001" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.9999998750000156 | 8000000                 |
      | lp2   | 0.0000001249999844 | 8000001                 |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |
    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"
    #liquidity fee: 20*1000*0.001=20
    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    # check lp fee distribution
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 19     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 0      | USD   |

  @FeeRound @NoPerp
  Scenario: 005: 2 LP distribution at settlement

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | lp3    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 8000000           | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 2970      | 1                    | buy  | BID              | 2970   | 2      |
      | lp1   | ETH/MAR22 | 5339      | 1                    | buy  | MID              | 5339   | 1      |
      | lp1   | ETH/MAR22 | 2420      | 1                    | sell | ASK              | 2420   | 2      |
      | lp1   | ETH/MAR22 | 5609      | 1                    | sell | MID              | 5609   | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 1                 | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | ETH/MAR22 | 1         | 1                    | buy  | BID              | 1          | 2      |
      | lp2   | ETH/MAR22 | 1         | 1                    | buy  | MID              | 1          | 1      |
      | lp2   | ETH/MAR22 | 1         | 1                    | sell | ASK              | 1          | 2      |
      | lp2   | ETH/MAR22 | 1         | 1                    | sell | MID              | 1          | 1      |
 
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the mark price should be "1000" for the market "ETH/MAR22"
    And the open interest should be "60" for the market "ETH/MAR22"
    And the target stake should be "6000" for the market "ETH/MAR22"
    And the supplied stake should be "8000001" for the market "ETH/MAR22"

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.9999998750000156 | 8000000                 |
      | lp2   | 0.0000001249999844 | 8000001                 |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |     
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    # settle the market
    When the oracles broadcast data signed with "0xDEADBEEF":
      | name               | value |
      | trading.terminated | true  |

    Then the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 1200  |

    # check lp fee distribution at s

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 19     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/MAR22 | 0      | USD   |

  Scenario: 007 3 LPs joining at start, unequal commitments, check Liquidity fee factors are recalculated every time the liquidity demand estimate changes.(0042-LIQF-005)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | lp3    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 2000              | 0.002 | submission |
      | lp1 | lp1   | ETH/MAR22 | 2000              | 0.002 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp1   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1          | 2      |
      | lp1   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1          | 1      |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 3000              | 0.003 | submission |
      | lp2 | lp2   | ETH/MAR22 | 3000              | 0.003 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp2   | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1          | 2      |
      | lp2   | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1          | 1      |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp3 | lp3   | ETH/MAR22 | 4000              | 0.004 | submission |
      | lp3 | lp3   | ETH/MAR22 | 4000              | 0.004 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lp3     | ETH/MAR22 | 2         | 1                    | sell | ASK              | 1          | 2      |
      | lp3     | ETH/MAR22 | 2         | 1                    | buy  | BID              | 1          | 1      |
 
    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"
    And the liquidity fee factor should be "0.002" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"
    # no fees in auction

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 25     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 25     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the accumulated liquidity fees should be "50" for the market "ETH/MAR22"
    #liquidity fee: 25*1000*0.002=50
    And the liquidity fee factor should be "0.003" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 25     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 25     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    And the accumulated liquidity fees should be "125" for the market "ETH/MAR22"
    #liquidity fee: 25*1000*0.003=75
    And the liquidity fee factor should be "0.004" for the market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 25   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 6000         | 9000           | 60            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.2222222222222222 | 2000                    |
      | lp2   | 0.3333333333333333 | 5000                    |
      | lp3   | 0.4444444444444444 | 9000                    |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the liquidity fee factor should be "0.004" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "325" for the market "ETH/MAR22"
    #liquidity fee: 50*1000*0.004=200, accumulated liquidity fee: 200+125=325

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 6000         | 9000           | 10            |

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 200    | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | sell | 200    | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the liquidity fee factor should be "0.004" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "1125" for the market "ETH/MAR22"
    #liquidity fee: 200*1000*0.004=800, accumulated liquidity fee: 800+325=1125

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 21000        | 9000           | 210           |


