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
    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | oracle config          |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |

    Given the average block duration is "2"

  @VirtStake
  Scenario: 001 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades (0042-LIQF-001)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment  |

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
      | buy  | 898   | 75     |
      | buy  | 900   | 1      |
      | buy  | 999   | 14     |
      | sell | 1102  | 61     |
      | sell | 1100  | 1      |
      | sell | 1001  | 14     |

    #volume = ceiling(liquidity_obligation x liquidity-normalised-proportion / probability_of_trading / price)
    #for any price better than the bid price or better than the ask price it returns 0.5
    #for any price in within 500 price ticks from the best bid/ask (i.e. worse than) it returns the probability as returned by the risk model (in this case 0.1 scaled by 0.5.
    #priceLvel at 898:10000*(1/3)/0.05/898=74.23
    #priceLvel at 999:10000*(2/3)/0.5/999=13.34
    #priceLvel at 1102:10000*(1/3)/0.05/1102=60.49
    #priceLvel at 1001:10000*(2/3)/0.5/1001=13.32

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 10000                   |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 10680  | 999979320 | 10000 |
      | party1 | USD   | ETH/MAR22 | 2758   | 99997242  | 0     |
      | party2 | USD   | ETH/MAR22 | 2652   | 99997348  | 0     |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should "0.001" for the market "ETH/MAR22"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 11640  | 999977631 | 10000 |
      | party1 | USD   | ETH/MAR22 | 1800   | 99998202  | 0     |
      | party2 | USD   | ETH/MAR22 | 1812   | 99998875  | 0     |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 75     |
      | buy  | 900   | 1      |
      | buy  | 1000  | 0      |
      | sell | 1000  | 15     |
      | sell | 1001  | 0      |
      | sell | 1102  | 0      |
      | sell | 1100  | 1      |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    And the accumulated liquidity fees should be "20" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    Then time is updated to "2019-11-30T00:20:05Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    # here we get only a trade for a volume of 15 as it's what was on the LP
    # order, then the 25 remaining from party1 are cancelled for self trade
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 15   | lp1    |

    # this is slightly different than expected, as the trades happen against the LP,
    # which is probably not what you expected initially
    And the accumulated liquidity fees should be "15" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:30:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 15     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

  @VirtStake
  Scenario: 002 2 LPs joining at start, equal commitments (0042-LIQF-002)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | sell | MID              | 2          | 1      | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | MID              | 2          | 1      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | MID              | 2          | 1      | amendment  |

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
      | lp1   | 0.5               | 10000                   |
      | lp2   | 0.5               | 10000                   |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should "0.002" for the market "ETH/MAR22"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 8    | lp1    |
      | party2 | 951   | 8    | lp2    |
      | party2 | 1000  | 4    | party1 |

    And the accumulated liquidity fees should be "40" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 40     | 1100  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/MAR22 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 8    | lp1    |
      | party1 | 951   | 8    | lp2    |

    And the accumulated liquidity fees should be "32" for the market "ETH/MAR22"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:08Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 16     | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 16     | USD   |

  @VirtStake
  Scenario: 003 2 LPs joining at start, unequal commitments (0042-LIQF-003)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 8000              | 0.001 | sell | MID              | 2          | 1      | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | buy  | MID              | 2          | 1      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.002 | sell | MID              | 2          | 1      | amendment  |

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

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 6000         | 10000          | 60            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 10000                   |
      | lp2   | 0.2               | 10000                   |

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 75     |
      | buy  | 900   | 1      |
      | buy  | 999   | 14     |
      | sell | 1102  | 62     |
      | sell | 1100  | 1      |
      | sell | 1001  | 14     |

    Then time is updated to "2019-11-30T00:20:10Z"
    #week2
  
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 5000              | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 6506         | 7000           | 65            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.6097560975609760   | 4268                    |
      | lp2   | 0.3902439024390240   | 2732                    |

    Then time is updated to "2019-11-30T04:20:10Z"
    #week3

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 53     |
      | buy  | 900   | 1      |
      | buy  | 999   | 10     |
      | sell | 1102  | 44     |
      | sell | 1100  | 1      |
      | sell | 1001  | 10     |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/MAR22 | sell | 3      | 999   | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 1980              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 1980              | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 501       | 1501      | 6793         | 6980           | 68            |
      # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 999 x 68 x 1 x 0.1

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7183701617769610   | 5014                    |
      | lp2   | 0.2816298382230400   | 1966                    |

    Then time is updated to "2019-11-30T05:22:10Z"
    #week4

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 53     |
      | buy  | 900   | 1      |
      | buy  | 999   | 10     |
      | sell | 1102  | 43     |
      | sell | 1100  | 1      |
      | sell | 1001  | 10     |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | sell | 2      | 999   | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 1880              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 1880              | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 500       | 1498      | 6793         | 6880           | 66            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7369141904364910   | 5070                    |
      | lp2   | 0.2630858095635090   | 1810                    |

    Then time is updated to "2019-11-30T08:22:10Z"
    #week5

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 52     |
      | buy  | 900   | 1      |
      | buy  | 999   | 10     |
      | sell | 1102  | 43     |
      | sell | 1100  | 1      |
      | sell | 1001  | 10     |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | sell | 2      | 999   | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 500       | 1498      | 6793         | 6880           | 64            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7267441860465116   | 5000                    |
      | lp2   | 0.2732558139534884   | 1880                    |

    Then time is updated to "2019-11-30T10:22:10Z"
    #week6
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 500       | 1498      | 6793         | 7000           | 64            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      #| lp1   | 0.7182132526872552   | 10000                   |
      #| lp2   | 0.2817867473127448   | 421.3495683743868003    |
      | lp1   | 0.7022471910112360   | 4916                    |
      | lp2   | 0.2977528089887640   | 2084                    |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 53     |
      | buy  | 900   | 1      |
      | buy  | 999   | 10     |
      | sell | 1102  | 44     |
      | sell | 1100  | 1      |
      | sell | 1001  | 10     |

    Then time is updated to "2019-11-30T12:22:10Z"
     #week7

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | sell | 1      | 999   | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 500       | 1498      | 6793         | 8000           | 63            |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      #| lp1   | 0.6552177177520024   | 10000                   |
      #| lp2   | 0.3447822822479976   | 2666.4086478022710885   |
      | lp1   | 0.5555555555555560   | 4444                    |
      | lp2   | 0.4444444444444440   | 3556                    |

