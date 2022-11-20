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
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config          |
      | ETH/MAR22 | USD        | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future |

    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 0s    |

    # block duration of 2 seconds
    And the average block duration is "2"

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

    And the liquidity fee factor should be "0.001" for the market "ETH/MAR22"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/MAR22 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | lp1    | USD   | ETH/MAR22 | 11787  | 999977484 | 10000 |
      #| lp1    | USD   | ETH/MAR22 | 12522  | 999976749 | 10000 |
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
    # network should move ahead 301 blocks -> 602s, or a good 10 minutes
    When the network moves ahead "301" blocks
    #Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 20     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    # move a good 10 minutes in time
    When the network moves ahead "301" blocks
    #Then time is updated to "2019-11-30T00:20:05Z"

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
    #When the network moves ahead "1" blocks
    When the network moves ahead "301" blocks
    #Then time is updated to "2019-11-30T00:30:05Z"

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
      | lp1   | 0.5               | 5000                    |
      | lp2   | 0.5               | 10000                   |

    And the price monitoring bounds for the market "ETH/MAR22" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.002" for the market "ETH/MAR22"

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
    When the network moves ahead "301" blocks

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
    When the network moves ahead "301" blocks

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
    # set default block duration to a bit more than half hour, hence 2 blocks will make a bit more than a timewindow (1h)
    And the average block duration is "1801"

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
      | lp1   | 0.8               | 8000                    |
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

    #timewindow1
    Then the network moves ahead "2" blocks
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

    #Since lp1 wants to decrease their commitment by delta < 0. Then we update: LP i virtual stake <- LP i virtual stake x (LP i stake + delta)/(LP i stake).
    #grwoth factor for this time window is 0 since there is no fee during auction in the previous timewindow, so taking growth factor into the calculation, lp1 virtual stake stays at 5000, lp2 virtual stake stays at 2000
    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7142857142857143   | 8000                    |
      | lp2   | 0.2857142857142857   | 10000                   |

    #timewindow2
    Then the network moves ahead "2" blocks
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
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 6793         | 6980           | 68            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 999 x 68 x 1 x 0.1

    #Since lp1 wants to decrease their commitment by delta < 0. Then we update: LP i virtual stake <- LP i virtual stake x (LP i stake + delta)/(LP i stake).
    #grwoth factor for this time window is -0.2, so taking growth factor into the calculation, LP i virtual stake <- max(LP i physical stake, (1 + r) x (LP i virtual stake)), lp1 virtual stake stays at 5000, lp2 virtual stake stays at 1980
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7163323782234957   | 8000                    |
      | lp2   | 0.2836676217765043   | 10000                   |

    #timewindow3
    Then the network moves ahead "2" blocks

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

    #Since lp2 wants to decrease their commitment by delta < 0. Then we update: LP i virtual stake <- LP i virtual stake x (LP i stake + delta)/(LP i stake).
    #grwoth factor for this time window is -0.17, so taking growth factor into the calculation, LP i virtual stake <- max(LP i physical stake, (1 + r) x (LP i virtual stake)), lp1 virtual stake stays at 5000, lp2 virtual stake stays at 1880
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7267441860465116   | 8000                    |
      | lp2   | 0.2732558139534884   | 10000                   |

    #Then time is updated to "2019-11-30T08:22:10Z"
    #timewindow4
    Then the network moves ahead "2" blocks
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

    #grwoth factor for this time window is -0.1, no change on commitment, so no change on virtual stake
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7267441860465116   | 8000                    |
      | lp2   | 0.2732558139534884   | 10000                   |

    #Then time is updated to "2019-11-30T10:22:10Z"
    #timewindow5
    Then the network moves ahead "2" blocks
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 2000              | 0.001 | sell | MID              | 2          | 1      | amendment  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 999        | TRADING_MODE_CONTINUOUS | 1       | 500       | 1498      | 6793         | 7000           | 64            |

    #lp2 wants to increase stake, we use this formula for lp2 virtual satke: LP i virtual stake <- LP i virtual stake + delta.
    #growth rate is -0.2
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.7142857142857143   | 8000                    |
      | lp2   | 0.2857142857142857   | 9820                    |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 53     |
      | buy  | 900   | 1      |
      | buy  | 999   | 10     |
      | sell | 1102  | 44     |
      | sell | 1100  | 1      |
      | sell | 1001  | 10     |

    #Then time is updated to "2019-11-30T12:22:10Z"
    #timewindow6
    Then the network moves ahead "2" blocks
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

    #lp2 wants to increase stake, we use this formula for lp2 virtual satke: LP i virtual stake <- LP i virtual stake + delta.
    #growth rate is -0.1
    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share    | average entry valuation |
      | lp1   | 0.625                | 8000                    |
      | lp2   | 0.375                | 9213.3333333333333334   |

  @VirtStake
  Scenario: 004 2 LPs joining at start, unequal commitments. Checking calculation of equity-like-shares and liquidity-fee-distribution in a shrinking market. (0042-LIQF-008 0042-LIQF-011)

    # Scenario has 6 market periods:

    # - 0th period (bootstrap period): no LP changes, no trades
    # - 1st period: 1 LPs decrease commitment, some trades occur
    # - 2nd period: 1 LPs increase commitment, some trades occur
    # - 3rd period: 2 LPs decrease commitment, some trades occur
    # - 4th period: 2 LPs increase commitment, some trades occur
    # - 5th period: 1 LPs decrease commitment, 1 LPs increase commitment, some trades occur


    # Scenario moves ahead to next market period by:

    # - moving ahead "1" blocks to trigger the next liquidity distribution
    # - moving ahead "1" blocks to trigger the next market period


    # Following checks occur in each market where trades:

    # - Check transfers from the price taker to the market-liquidity-pool are correct
    # - Check accumulated-liquidity-fees are non-zero and correct
    # - Check equity-like-shares are correct
    # - Check transfers from the market-liquidity-pool to the liquidity-providers are correct
    # - Check accumulated-liquidity-fees are zero

    Given the average block duration is "1801"

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | lp1    | USD   | 100000 |
      | lp2    | USD   | 100000 |
      | party1 | USD   | 100000 |
      | party2 | USD   | 100000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      | amendment  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | MID              | 3          | 1      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | MID              | 3          | 1      | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |


    # 0th period (bootstrap period): no LP changes, no trades
    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 50   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 5000         | 10000          | 50            |

    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 57     |
      | buy  | 900   | 1      |
      | buy  | 999   | 17     |
      | sell | 1001  | 15     |
      | sell | 1100  | 1      |
      | sell | 1102  | 47     |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 4000                    |
      | lp2   | 0.6               | 10000                   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 1st period: 1 LPs decrease commitment, some trades occur:
    When the network moves ahead "2" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 2      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 2    | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 2 * 0.002) = ceil(4.004) = 5

    And the following transfers should happen:
      | from   | to     | from account           | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL   | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 5      | USD   |

    And the accumulated liquidity fees should be "5" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 500       | 1500      | 5205         | 9000           | 52            |

    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 51     |
      | buy  | 900   | 1      |
      | buy  | 999   | 15     |
      | sell | 1001  | 14     |
      | sell | 1100  | 1      |
      | sell | 1102  | 42     |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.3333333333333333 | 4000                    |
      | lp2   | 0.6666666666666667 | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 1      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 4      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 2nd period: 1 LPs increase commitment, some trades occur
    When the network moves ahead "1" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 2      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 2    | lp2    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 2 * 0.002) = ceil(4.004) = 5

    And the following transfers should happen:
      | from   | to     | from account           | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL   | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 5      | USD   |

    And the accumulated liquidity fees should be "5" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 5405         | 10000          | 54            |

    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 57     |
      | buy  | 900   | 1      |
      | buy  | 999   | 17     |
      | sell | 1001  | 15     |
      | sell | 1100  | 1      |
      | sell | 1102  | 47     |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share  | average entry valuation |
      | lp1   | 0.4                | 5500                    |
      | lp2   | 0.6                | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 3rd period: 2 LPs decrease commitment, some trades occur
    When the network moves ahead "1" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 5000              | 0.002 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 3      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 3    | lp1    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 3 * 0.002) = ceil(6.006) = 7

    And the following transfers should happen:
      | from   | to     | from account           | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL   | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 7      | USD   |

    And the accumulated liquidity fees should be "7" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 5705         | 8000           | 57            |

    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 45     |
      | buy  | 900   | 1      |
      | buy  | 999   | 13     |
      | sell | 1001  | 13     |
      | sell | 1100  | 1      |
      | sell | 1102  | 37     |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.375             | 5500                    |
      | lp2   | 0.625             | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 2      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 5      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 4nd period: 2 LPs increase commitment, some trades occur
    When the network moves ahead "2" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 4000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 6000              | 0.002 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 4      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 4    | lp1    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 4 * 0.002) = ceil(8.008) = 9

    And the following transfers should happen:
      | from   | to     | from account           | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL   | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 9      | USD   |

    And the accumulated liquidity fees should be "9" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 6106         | 10000          | 61            |

    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 57     |
      | buy  | 900   | 1      |
      | buy  | 999   | 17     |
      | sell | 1001  | 15     |
      | sell | 1100  | 1      |
      | sell | 1102  | 47     |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share   | average entry valuation |
      | lp1   | 0.4                 | 6375                    |
      | lp2   | 0.6                 | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 6      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 5th period: 1 LPs decrease commitment 1 LPs increase commitment, some trades occur
    When the network moves ahead "1" blocks:

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | BID              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | buy  | MID              | 3          | 1      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | ASK              | 1          | 2      | amendment |
      | lp1 | lp1   | ETH/MAR22 | 3000              | 0.001 | sell | MID              | 3          | 1      | amendment |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | buy  | BID              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | buy  | MID              | 3          | 1      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | sell | ASK              | 1          | 2      | amendment |
      | lp2 | lp2   | ETH/MAR22 | 7000              | 0.002 | sell | MID              | 3          | 1      | amendment |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 5      | 1001  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1001  | 5    | lp1    |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1001 * 11 * 0.002) = ceil(10.01) = 11

    And the following transfers should happen:
      | from   | to     | from account           | to account                  | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_GENERAL   | ACCOUNT_TYPE_FEES_LIQUIDITY | ETH/MAR22 | 11     | USD   |

    And the accumulated liquidity fees should be "11" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1001       | TRADING_MODE_CONTINUOUS | 1       | 502       | 1500      | 6606         | 10000          | 66            |

    And the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | buy  | 898   | 56     |
      | buy  | 900   | 1      |
      | buy  | 999   | 16     |
      | sell | 1001  | 16     |
      | sell | 1100  | 1      |
      | sell | 1102  | 46     |

    # Trigger next liquidity fee distribution without triggering next period
    When the network moves ahead "1" blocks:

    Then the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share   | average entry valuation |
      | lp1   | 0.3                 | 6375                    |
      | lp2   | 0.7                 | 10000                   |

    And the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 3      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 8      | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"

  @VirtStake
  Scenario: 005 2 LPs joining at start, 1 LP forcibly closed out (0042-LIQF-008)

    Given the average block duration is "601"

    When the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | lp1    | USD   | 10000    |
      | lp2    | USD   | 10000000 |
      | party1 | USD   | 10000000 |
      | party2 | USD   | 10000000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 1000              | 0.001 | buy  | BID              | 1          | 51     | submission |
      | lp1 | lp1   | ETH/MAR22 | 1000              | 0.001 | sell | ASK              | 1          | 51     | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/MAR22 | 9000              | 0.002 | buy  | BID             | 1           | 51     | submission |
      | lp2 | lp2   | ETH/MAR22 | 9000              | 0.002 | sell | ASK             | 1           | 51     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | pa1-b1    |
      | party1 | ETH/MAR22 | buy  | 15     | 950   | 0                | TYPE_LIMIT | TIF_GTC | pa1-b2    |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | pa2-s1    |
      | lp1    | ETH/MAR22 | sell | 15     | 950   | 0                | TYPE_LIMIT | TIF_GTC | lp1-s1    |

    Then the opening auction period ends for market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 7308   | 1692    | 1000 |


    # 1st set of trades: market moves against lp1s position, margin-insufficient, margin topped up from general and bond
    When the network moves ahead "1" blocks:

    And the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party1 | pa1-b1    | 1050  | 0          | TIF_GTC |
      | party2 | pa2-s1    | 1250  | 0          | TIF_GTC |

    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 30     | 1150  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 30     | 1150  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1150  | 30   | party2 |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1150 * 30 * 0.002) = ceil(69) = 69

    And the accumulated liquidity fees should be "69" for the market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 6924   | 0       | 76   |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.1               | 1000                    |
      | lp2   | 0.9               | 10000                   |

    # Trigger liquidity distribution
    When the network moves ahead "1" blocks:

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 6      | USD   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 63     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"


    # 2nd set of trades: market moves against LP1s position, margin-insufficient, position partly closed out
    When the network moves ahead "1" blocks:

    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     |
      | party1 | pa1-b1    | 1200  | 0          | TIF_GTC |
      | party2 | pa2-s1    | 1400  | 0          | TIF_GTC |

    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 30     | 1300  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 30     | 1300  | 1                | TYPE_LIMIT | TIF_GTC |

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1300  | 30   | party2 |

    # liquidity_fee = ceil(volume * price * liquidity_fee_factor) =  ceil(1300 * 30 * 0.002) = ceil(78) = 78

    And the accumulated liquidity fees should be "78" for the market "ETH/MAR22"

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | bond |
      | lp1   | USD   | ETH/MAR22 | 4756   | 0       | 0    |

    And the liquidity provider fee shares for the market "ETH/MAR22" should be:
      | party | equity like share | average entry valuation |
      | lp2   | 1                 | 10000                   |

    # Trigger liquidity distribution
    When the network moves ahead "1" blocks:

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/MAR22 | 78     | USD   |

    And the accumulated liquidity fees should be "0" for the market "ETH/MAR22"
