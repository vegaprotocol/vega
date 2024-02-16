Feature: Test liquidity provider reward distribution

  # Spec file: ../spec/0042-setting-fees-and-rewarding-lps.md

  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.00             | 24h         | 1              |  
      
    And the following network parameters are set:
      | name                                        | value |
      | market.value.windowLength                   | 1h    |
      | network.markPriceUpdateMaximumFrequency     | 1s    |
      | network.markPriceUpdateMaximumFrequency     | 0s    |
      | limits.markets.maxPeggedOrders              | 612   |
      | market.liquidity.equityLikeShareFeeFraction | 1     |
    And the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 3                 |
    And the price monitoring named "price-monitoring-2":
      | horizon | probability | auction extension |
      | 10000   | 0.9999999   | 3                 |
    And the oracle spec for settlement data filtering data from "0xCAFECAFE" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model             | margin calculator         | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/DEC21 | ETH        | ETH   | lqm-params           | simple-risk-model-1    | default-margin-calculator | 2                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 1e0                    | 1e0                       | SLA        |
      | ETH/DEC22 | ETH        | ETH   | lqm-params           | lognormal-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring-2 | ethDec21Oracle     | 1e0                    | 1e0                       | SLA        |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 600s  |
    And the average block duration is "1"

  Scenario: 001, 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 15        | 10                   | buy  | BID              | 30     | 2      |
      | lp1   | ETH/DEC21 | 15        | 10                   | buy  | MID              | 30     | 1      |
      | lp1   | ETH/DEC21 | 15        | 10                   | sell | ASK              | 30     | 2      |
      | lp1   | ETH/DEC21 | 15        | 10                   | sell | MID              | 30     | 1      |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "10" for the market "ETH/DEC21"
    And the target stake should be "1000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 1                 | 10000                   |

    Then the network moves ahead "1" blocks

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    Then the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 20     | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then time is updated to "2019-11-30T00:20:05Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/DEC21 | buy  | 8      | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy |
    #   | party2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    # this is slightly different than expected, as the trades happen against the LP,
    # which is probably not what you expected initially
    And the accumulated liquidity fees should be "8" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:30:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 8      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


  Scenario: 002, 2 LPs joining at start, equal commitments

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 20        | 15                   | buy  | BID              | 100    | 2      |
      | lp1   | ETH/DEC21 | 20        | 15                   | buy  | MID              | 200    | 1      |
      | lp1   | ETH/DEC21 | 20        | 15                   | sell | ASK              | 100    | 2      |
      | lp1   | ETH/DEC21 | 20        | 15                   | sell | MID              | 100    | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | submission |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | amendment  |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | amendment  |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 20        | 15                   | buy  | BID              | 100    | 2      |
      | lp2   | ETH/DEC21 | 20        | 15                   | buy  | MID              | 100    | 1      |
      | lp2   | ETH/DEC21 | 20        | 15                   | sell | ASK              | 100    | 2      |
      | lp2   | ETH/DEC21 | 20        | 15                   | sell | MID              | 100    | 1      |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |


    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "90" for the market "ETH/DEC21"
    And the target stake should be "9000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 5000                    |
      | lp2   | 0.5               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the accumulated liquidity fees should be "39" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 19     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 19     | ETH   |


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1100  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the accumulated liquidity fees should be "79" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:08Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 39     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 39     | ETH   |

  @FeeRound
  Scenario: 003, 2 LPs joining at start, equal commitments, unequal offsets
    Given the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 10    |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |
    And the liquidity sla params named "updated-SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets are updated:
      | id        | sla params  | linear slippage factor | quadratic slippage factor |
      | ETH/DEC22 | updated-SLA | 1e0                    | 1e0                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | ETH   | 100000000000 |
      | lp2    | ETH   | 100000000000 |
      | party1 | ETH   | 100000000    |
      | party2 | ETH   | 100000000    |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC22 | 40000             | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC22 | 40000             | 0.001 |            |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 | submission |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 |            |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference  |
      | lp1   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 32     | lp1-bids-1 |
      | lp1   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 32     | lp1-asks-1 |
      | lp2   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 102    | lp2-bids-1 |
      | lp2   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 102    | lp2-asks-1 |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC22 | buy  | 1      | 995   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | sell | 1      | 1005  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC22"
    Then the market data for the market "ETH/DEC22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000   | 893       | 1120      | 43908        | 80000          | 90            | 995                   | 1000             | 1005                    |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |
    And the liquidity provider fee shares for the market "ETH/DEC22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 40000                   |
      | lp2   | 0.5               | 80000                   |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks

    Then the accumulated liquidity fees should be "1" for the market "ETH/DEC22"


    # observe that lp2 gets lower share of the fees despite the same commitment amount (that is due to their orders being much wider than those of lp1)
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 26     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 13     | ETH   |

    # modify lp2 orders so that they fall outside the price monitoring bounds
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference  |
      | lp2   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 116    | lp2-bids-2 |
      | lp2   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 116    | lp2-asks-2 |
    And the parties cancel the following orders:
      | party | reference  |
      | lp2   | lp2-bids-1 |
      | lp2   | lp2-asks-1 |
    And the market data for the market "ETH/DEC22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000   | 893       | 1120      | 995                   | 1000             | 1005                    |
    And the accumulated liquidity fees should be "1" for the market "ETH/DEC22"



    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "41" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks

    # all the fees go to lp2
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 41     | ETH   |
      # zero fee shares are filtered out now, so no transfer events are generated
      #| market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 0      | ETH   |

    # lp2 manually adds some limit orders within PM range, observe automatically deployed orders go down and fee share go up
    When clear all events
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp2   | ETH/DEC22 | buy  | 100    | 995   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | ETH/DEC22 | sell | 100    | 1005  | 0                | TYPE_LIMIT | TIF_GTC |


    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # lp2 has increased their liquidity score by placing limit orders closer to the mid (and within price monitoring bounds),
    # hence their fee share is larger (and no longer 0) now.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 24     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 15     | ETH   |

    # lp2 manually adds some pegged orders within PM range, liquidity obligation is now fullfiled by limit and pegged orders so no automatic order deployment takes place
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference  |
      | lp2   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 1      | lp2-bids-3 |
      | lp2   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 1      | lp2-asks-3 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "41" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # lp2 has increased their liquidity score by placing pegged orders closer to the mid (and within price monitoring bounds),
    # hence their fee share is larger now.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 22     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 18     | ETH   |

  @FeeRound
  Scenario: 004, 2 LPs joining at start, unequal commitments

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 100       | 1                    | buy  | BID              | 200    | 2      |
      | lp1   | ETH/DEC21 | 100       | 1                    | sell | ASK              | 200    | 2      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 100       | 1                    | buy  | BID              | 200    | 2      |
      | lp2   | ETH/DEC21 | 100       | 1                    | sell | ASK              | 200    | 2      |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 8000                    |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 1000  | 20   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"
    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 40   | party2 |

    And the accumulated liquidity fees should be "40" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 32     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 8      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  @FeeRound
  Scenario: 005, 2 LPs joining at start, unequal commitments, 1 LP lp3 joining later, and 4 LPs lp4/5/6/7 with large commitment and low/high fee joins later (0042-LIQF-032)

    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 600s  |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | ETH   | 100000000000 |
      | lp2    | ETH   | 100000000000 |
      | lp3    | ETH   | 100000000000 |
      | lp4    | ETH   | 100000000000 |
      | lp5    | ETH   | 100000000000 |
      | lp6    | ETH   | 100000000000 |
      | lp7    | ETH   | 100000000000 |
      | party1 | ETH   | 100000000000 |
      | party2 | ETH   | 100000000000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp1   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp2   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1000   | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1000   | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 8000                    |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 1000  | 20   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:15:00Z"
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp3   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp3   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "1" blocks"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 8000                    |
      | lp2   | 0.1               | 10000                   |
      | lp3   | 0.5               | 20000                   |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:20:00Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 16     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 16     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 16   | party2 |

    And the accumulated liquidity fees should be "16" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:20:06Z"
    # lp3 gets lower fee share than indicated by the ELS this fee round as it was later to deploy liquidity (so lower liquidity scores than others had)
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 8      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 2      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 5      | ETH   |

    And the accumulated liquidity fees should be "1" for the market "ETH/DEC21"

    # make sure we're in the next time window now
    When time is updated to "2019-11-30T00:30:07Z"
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 16     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 16     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "17" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:40:08Z"
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 6      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 1      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 8      | ETH   |
    And the accumulated liquidity fees should be "2" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    And the target stake should be "7200" for the market "ETH/DEC21"
    And the supplied stake should be "20000" for the market "ETH/DEC21"
     #AC 0042-LIQF-024:lp4 joining a market that is above the target stake with a commitment large enough to push one of two higher bids above the target stake, and a higher fee bid than the current fee: the fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp4 | lp4   | ETH/DEC21 | 20000             | 0.004 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp4   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp4   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "2" blocks
    When time is updated to "2019-11-30T00:50:09Z"
    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    #AC 0042-LIQF-029: lp5 joining a market that is above the target stake with a sufficiently large commitment to push ALL higher bids above the target stake and a lower fee bid than the current fee: their fee is used
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp5 | lp5   | ETH/DEC21 | 30000             | 0.0005 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp5   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp5   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "2" blocks
    When time is updated to "2019-11-30T01:00:10Z"
    And the liquidity fee factor should be "0.0005" for the market "ETH/DEC21"
    And the target stake should be "7200" for the market "ETH/DEC21"
    And the supplied stake should be "70000" for the market "ETH/DEC21"

    #AC 0042-LIQF-030: lp6 joining a market that is above the target stake with a commitment not large enough to push any higher bids above the target stake, and a lower fee bid than the current fee: the fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp6 | lp6   | ETH/DEC21 | 2000              | 0.0001 | submission |
      | lp6 | lp6   | ETH/DEC21 | 2000              | 0.0001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp6   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp6   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "2" blocks
    When time is updated to "2019-11-30T01:10:11Z"
    And the liquidity fee factor should be "0.0005" for the market "ETH/DEC21"

  @FeeRound
  Scenario: 006, 2 LPs joining at start, unequal commitments, market settles (0042-LIQF-014)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp1   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp2   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 1000  | 20   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |


    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 40   | party2 |

    And the accumulated liquidity fees should be "40" for the market "ETH/DEC21"

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name               | value |
      | trading.terminated | true  |
    Then the market state should be "STATE_TRADING_TERMINATED" for the market "ETH/DEC21"

    When the oracles broadcast data signed with "0xCAFECAFE":
      | name             | value |
      | prices.ETH.value | 42    |
    Then the market state should be "STATE_SETTLED" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
    And the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 32     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 8      | ETH   |

  Scenario: 007, 2 LPs joining at start, unequal commitments, 1 leaves later (0042-LIQF-012)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp1   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp2   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 8000                    |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 1000  | 20   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"
    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 40   | party2 |

    And the accumulated liquidity fees should be "40" for the market "ETH/DEC21"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    Then the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status        |
      | lp2 | lp2   | ETH/DEC21 | 2000              | STATUS_ACTIVE |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type      |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | cancellation |

    Then time is updated to "2019-11-30T00:20:06Z"

    # now all the accumulated fees go to the remaining lp (as the other one cancelled their provision)
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 32     | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  Scenario: 008, 2 LPs joining at start, unequal commitments, 1 LP joins later , and 1 LP leave

    Given the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 10    |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | limits.markets.maxPeggedOrders                   | 612   |
    #   | validators.epoch.length                          | 1s    |

    Given the liquidity sla params named "updated-SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.0                          | 1                             | 0.0                    |
    And the markets are updated:
      | id        | sla params  | linear slippage factor | quadratic slippage factor |
      | ETH/DEC22 | updated-SLA | 1e0                    | 1e0                       |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | lp3    | ETH   | 1000000000 |
      | lp4    | ETH   | 1000000000 |
      | lp5    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.002 | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 1      |
      | lp1   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 1      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.001 | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 1      |
      | lp2   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 1      |
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the target stake should be "9000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    And the network moves ahead "1" blocks

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp3 | lp3   | ETH/DEC21 | 9000              | 0.0015 | submission |
      | lp3 | lp3   | ETH/DEC21 | 9000              | 0.0015 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp3   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 4      |
      | lp3   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 4      |
    And the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0015" for the market "ETH/DEC21"
    And the network moves ahead "10" blocks

    And the target stake should be "9000" for the market "ETH/DEC21"
    And the supplied stake should be "19000" for the market "ETH/DEC21"

    #AC 0042-LIQF-025: lp3 leaves a market that is above target stake when their fee bid is currently being used: fee changes to fee bid by the LP who takes their place in the bidding order
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type      |
      | lp3 | lp3   | ETH/DEC21 | 9000              | 0.0015 | cancellation |
      | lp3 | lp3   | ETH/DEC21 | 9000              | 0.0015 | cancellation |
    And the network moves ahead "10" blocks
    And the target stake should be "9000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 30     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"
    And the network moves ahead "1" blocks
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"
    #AC 0042-LIQF-020: lp3 joining a market that is below the target stake with a lower fee bid than the current fee: fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp3 | lp3   | ETH/DEC21 | 1000              | 0.0001 | submission |
      | lp3 | lp3   | ETH/DEC21 | 1000              | 0.0001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp3   | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 4      |
      | lp3   | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 4      |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"
    And the network moves ahead "1" blocks
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "11000" for the market "ETH/DEC21"

    #AC 0042-LIQF-019: lp3 joining a market that is below the target stake with a higher fee bid than the current fee: their fee is used
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp3 | lp3   | ETH/DEC21 | 2000              | 0.003 | amendment |
      | lp3 | lp3   | ETH/DEC21 | 2000              | 0.003 | amendment |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp3   | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 4      |
      | lp3   | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 4      |
    And the network moves ahead "2" blocks
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "12000" for the market "ETH/DEC21"

    #lp4 join when market is below target stake with a large commitment
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp4 | lp4   | ETH/DEC21 | 10000             | 0.004 | submission |
      | lp4 | lp4   | ETH/DEC21 | 10000             | 0.004 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp4   | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 4      |
      | lp4   | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 4      |
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the network moves ahead "1" blocks

    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "22000" for the market "ETH/DEC21"

    # AC 0042-LIQF-028: lp4 leaves a market that is above target stake when their fee bid is higher than the one currently being used: fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type      |
      | lp4 | lp4   | ETH/DEC21 | 10000             | 0.004 | cancellation |
      | lp4 | lp4   | ETH/DEC21 | 10000             | 0.004 | cancellation |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp4   | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 4      |
      | lp4   | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 4      |
    And the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "12000" for the market "ETH/DEC21"
    # lp4 join when market is above target stake with a large commitment
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp4 | lp4   | ETH/DEC21 | 4000              | 0.004 | submission |
      | lp4 | lp4   | ETH/DEC21 | 4000              | 0.004 | submission |
    And the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "16000" for the market "ETH/DEC21"

    # AC 0042-LIQF-023: An LP joining a market that is above the target stake with a commitment large enough to push one of two higher bids above the target stake, and a lower fee bid than the current fee: the fee changes to the other lower bid
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp5 | lp5   | ETH/DEC21 | 6000              | 0.0015 | submission |
      | lp5 | lp5   | ETH/DEC21 | 6000              | 0.0015 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp5   | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 4      |
      | lp5   | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 4      |
    And the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "22000" for the market "ETH/DEC21"

    #AC 0042-LIQF-026: An LP leaves a market that is above target stake when their fee bid is lower than the one currently being used and their commitment size changes the LP that meets the target stake: fee changes to fee bid by the LP that is now at the place in the bid order to provide the target stake
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type      |
      | lp5 | lp5   | ETH/DEC21 | 6000              | 0.0015 | cancellation |
      | lp5 | lp5   | ETH/DEC21 | 6000              | 0.0015 | cancellation |
    And the network moves ahead "2" blocks
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "16000" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp5 | lp5   | ETH/DEC21 | 1000              | 0.0015 | submission |
      | lp5 | lp5   | ETH/DEC21 | 1000              | 0.0015 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp5   | ETH/DEC21 | 2         | 1                    | buy  | BID              | 1      | 4      |
      | lp5   | ETH/DEC21 | 2         | 1                    | sell | ASK              | 1      | 4      |
    And the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "17000" for the market "ETH/DEC21"

    #AC 0042-LIQF-027: An LP leaves a market that is above target stake when their fee bid is lower than the one currently being used and their commitment size doesn't change the LP that meets the target stake: fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type      |
      | lp5 | lp5   | ETH/DEC21 | 1000              | 0.0015 | cancellation |
      | lp5 | lp5   | ETH/DEC21 | 1000              | 0.0015 | cancellation |
    And the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.003" for the market "ETH/DEC21"
    And the target stake should be "12000" for the market "ETH/DEC21"
    And the supplied stake should be "16000" for the market "ETH/DEC21"

  @FeeRound
  Scenario: 005b, 2 LPs joining at start, unequal commitments, 1 LP lp3 joining later, and 4 LPs lp4/5/6/7 with large commitment and low/high fee joins later (0042-LIQF-032) with the fee factor set to 0.5

    Given the following network parameters are set:
      | name                                        | value |
      | market.liquidity.equityLikeShareFeeFraction | 0.5   |

    And the following network parameters are set:
      | name                    | value |
      | validators.epoch.length | 600s  |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | ETH   | 100000000000 |
      | lp2    | ETH   | 100000000000 |
      | lp3    | ETH   | 100000000000 |
      | lp4    | ETH   | 100000000000 |
      | lp5    | ETH   | 100000000000 |
      | lp6    | ETH   | 100000000000 |
      | lp7    | ETH   | 100000000000 |
      | party1 | ETH   | 100000000000 |
      | party2 | ETH   | 100000000000 |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp1   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp2   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp2   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 1000   | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC21 | buy  | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 1000   | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 60   | party2 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the mark price should be "1000" for the market "ETH/DEC21"
    And the open interest should be "60" for the market "ETH/DEC21"
    And the target stake should be "6000" for the market "ETH/DEC21"
    And the supplied stake should be "10000" for the market "ETH/DEC21"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.8               | 8000                    |
      | lp2   | 0.2               | 10000                   |

    And the price monitoring bounds for the market "ETH/DEC21" should be:
      | min bound | max bound |
      | 500       | 1500      |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 1000  | 20   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:15:00Z"
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp3   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp3   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "1" blocks"

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 8000                    |
      | lp2   | 0.1               | 10000                   |
      | lp3   | 0.5               | 20000                   |

    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:20:00Z"
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 16     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 16     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 16   | party2 |

    And the accumulated liquidity fees should be "16" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:20:06Z"
    # lp3 gets lower fee share than indicated by the ELS this fee round as it was later to deploy liquidity (so lower liquidity scores than others had)
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 7      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "1" for the market "ETH/DEC21"

    # make sure we're in the next time window now
    When time is updated to "2019-11-30T00:30:07Z"
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 16     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC21 | sell | 16     | 1000  | 1                | TYPE_LIMIT | TIF_GTC |
    Then the accumulated liquidity fees should be "17" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:40:08Z"
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 6      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 1      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC21 | 8      | ETH   |
    And the accumulated liquidity fees should be "2" for the market "ETH/DEC21"
    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    And the target stake should be "7200" for the market "ETH/DEC21"
    And the supplied stake should be "20000" for the market "ETH/DEC21"
     #AC 0042-LIQF-024:lp4 joining a market that is above the target stake with a commitment large enough to push one of two higher bids above the target stake, and a higher fee bid than the current fee: the fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp4 | lp4   | ETH/DEC21 | 20000             | 0.004 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp4   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp4   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "2" blocks
    When time is updated to "2019-11-30T00:50:09Z"
    And the liquidity fee factor should be "0.001" for the market "ETH/DEC21"
    #AC 0042-LIQF-029: lp5 joining a market that is above the target stake with a sufficiently large commitment to push ALL higher bids above the target stake and a lower fee bid than the current fee: their fee is used
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp5 | lp5   | ETH/DEC21 | 30000             | 0.0005 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp5   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp5   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "2" blocks
    When time is updated to "2019-11-30T01:00:10Z"
    And the liquidity fee factor should be "0.0005" for the market "ETH/DEC21"
    And the target stake should be "7200" for the market "ETH/DEC21"
    And the supplied stake should be "70000" for the market "ETH/DEC21"

    #AC 0042-LIQF-030: lp6 joining a market that is above the target stake with a commitment not large enough to push any higher bids above the target stake, and a lower fee bid than the current fee: the fee doesn't change
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type    |
      | lp6 | lp6   | ETH/DEC21 | 2000              | 0.0001 | submission |
      | lp6 | lp6   | ETH/DEC21 | 2000              | 0.0001 | submission |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp6   | ETH/DEC21 | 200       | 1                    | buy  | BID              | 200    | 2      |
      | lp6   | ETH/DEC21 | 200       | 1                    | sell | ASK              | 200    | 2      |
    Then the network moves ahead "2" blocks
    When time is updated to "2019-11-30T01:10:11Z"
    And the liquidity fee factor should be "0.0005" for the market "ETH/DEC21"

  @FeeRound
  Scenario: 003b, 2 LPs joining at start, equal commitments, unequal offsets with fee fraction set to 0.5
    Given the following network parameters are set:
      | name                                             | value |
      | limits.markets.maxPeggedOrders                   | 10    |
      | market.liquidity.providersFeeCalculationTimeStep | 5s    |
      | market.liquidity.equityLikeShareFeeFraction      | 0.5   |
    And the liquidity sla params named "updated-SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the markets are updated:
      | id        | sla params  | linear slippage factor | quadratic slippage factor |
      | ETH/DEC22 | updated-SLA | 1e0                    | 1e0                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | lp1    | ETH   | 100000000000 |
      | lp2    | ETH   | 100000000000 |
      | party1 | ETH   | 100000000    |
      | party2 | ETH   | 100000000    |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/DEC22 | 40000             | 0.001 | submission |
      | lp1 | lp1   | ETH/DEC22 | 40000             | 0.001 |            |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 | submission |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 |            |
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference  |
      | lp1   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 32     | lp1-bids-1 |
      | lp1   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 32     | lp1-asks-1 |
      | lp2   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 102    | lp2-bids-1 |
      | lp2   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 102    | lp2-asks-1 |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC22 | buy  | 1      | 995   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | buy  | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | sell | 1      | 1005  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | sell | 90     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
    And the opening auction period ends for market "ETH/DEC22"
    Then the market data for the market "ETH/DEC22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000   | 893       | 1120      | 43908        | 80000          | 90            | 995                   | 1000             | 1005                    |
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 90   | party2 |
    And the liquidity provider fee shares for the market "ETH/DEC22" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.5               | 40000                   |
      | lp2   | 0.5               | 80000                   |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks

    Then the accumulated liquidity fees should be "1" for the market "ETH/DEC22"


    # observe that lp2 gets lower share of the fees despite the same commitment amount (that is due to their orders being much wider than those of lp1)
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 26     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 13     | ETH   |

    # modify lp2 orders so that they fall outside the price monitoring bounds
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference  |
      | lp2   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 116    | lp2-bids-2 |
      | lp2   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 116    | lp2-asks-2 |
    And the parties cancel the following orders:
      | party | reference  |
      | lp2   | lp2-bids-1 |
      | lp2   | lp2-asks-1 |
    And the market data for the market "ETH/DEC22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000   | 893       | 1120      | 995                   | 1000             | 1005                    |
    And the accumulated liquidity fees should be "1" for the market "ETH/DEC22"



    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "41" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks

    # all the fees go to lp2
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 41     | ETH   |
      # 0 fee transfer is still filtered out to cut down on events.
      #| market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 0      | ETH   |

    # lp2 manually adds some limit orders within PM range, observe automatically deployed orders go down and fee share go up
    When clear all events
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp2   | ETH/DEC22 | buy  | 100    | 995   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp2   | ETH/DEC22 | sell | 100    | 1005  | 0                | TYPE_LIMIT | TIF_GTC |


    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # lp2 has increased their liquidity score by placing limit orders closer to the mid (and within price monitoring bounds),
    # hence their fee share is larger (and no longer 0) now.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 24     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 15     | ETH   |

    # lp2 manually adds some pegged orders within PM range, liquidity obligation is now fullfiled by limit and pegged orders so no automatic order deployment takes place
    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset | reference  |
      | lp2   | ETH/DEC22 | 100       | 50                   | buy  | BID              | 100    | 1      | lp2-bids-3 |
      | lp2   | ETH/DEC22 | 100       | 50                   | sell | ASK              | 100    | 1      | lp2-asks-3 |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "41" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # lp2 has increased their liquidity score by placing pegged orders closer to the mid (and within price monitoring bounds),
    # hence their fee share is larger now.
    Then the following transfers should happen:
      | from   | to  | from account                | to account                     | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 22     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_LP_LIQUIDITY_FEES | ETH/DEC22 | 18     | ETH   |


