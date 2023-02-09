Feature: Test liquidity provider reward distribution

  # Spec file: ../spec/0042-setting-fees-and-rewarding-lps.md

  Background:
    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
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
    And the following network parameters are set:
      | name                                                | value |
      | market.value.windowLength                           | 1h    |
      | market.stake.target.timeWindow                      | 24h   |
      | market.stake.target.scalingFactor                   | 1     |
      | market.liquidity.targetstake.triggering.ratio       | 0     |
      | market.liquidity.providers.fee.distributionTimeStep | 10m   |
      | network.markPriceUpdateMaximumFrequency             | 1s    |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
    And the markets:
      | id        | quote name | asset | risk model             | margin calculator         | auction duration | fees          | price monitoring   | data source config | linear slippage factor | quadratic slippage factor |
      | ETH/DEC21 | ETH        | ETH   | simple-risk-model-1    | default-margin-calculator | 2                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 1e6                    | 1e6                       |
      | ETH/DEC22 | ETH        | ETH   | lognormal-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring-2 | ethDec21Oracle     | 1e6                    | 1e6                       |
    And the average block duration is "1"

  Scenario: 1 LP joining at start, checking liquidity rewards over 3 periods, 1 period with no trades
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | amendment  |

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
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"
    Then time is updated to "2019-11-30T00:20:05Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1100  | 1                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    # here we get only a trade for a volume of 15 as it's what was on the LP
    # order, then the 25 remaining from party1 are cancelled for self trade
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 8    | lp1    |

    # this is slightly different than expected, as the trades happen against the LP,
    # which is probably not what you expected initially
    And the accumulated liquidity fees should be "8" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:30:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 8      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"


  Scenario: 2 LPs joining at start, equal commitments

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 5000              | 0.001 | sell | MID              | 2          | 1      | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | buy  | MID              | 2          | 1      | amendment  |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/DEC21 | 5000              | 0.002 | sell | MID              | 2          | 1      | amendment  |

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
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 4    | lp1    |
      | party2 | 951   | 4    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "40" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 20     | ETH   |


    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1100  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1100  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 4    | lp1    |
      | party1 | 951   | 4    | lp2    |

    And the accumulated liquidity fees should be "16" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:08Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 8      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 8      | ETH   |

  Scenario: 2 LPs joining at start, equal commitments, unequal offsets
    Given the following network parameters are set:
      | name                                                | value |
      | market.liquidity.providers.fee.distributionTimeStep | 5s    |
      | limits.markets.maxPeggedOrders                      | 10    |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC22 | 40000             | 0.001 | buy  | BID              | 1          | 32     | submission |
      | lp1 | lp1   | ETH/DEC22 | 40000             | 0.001 | sell | ASK              | 1          | 32     |            |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 | buy  | BID              | 1          | 102    | submission |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 | sell | ASK              | 1          | 102    |            |

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
    And the orders should have the following states:
      | party | market id | side | volume | price | status        | reference |
      | lp1   | ETH/DEC22 | buy  | 42     | 963   | STATUS_ACTIVE | lp1       |
      | lp1   | ETH/DEC22 | sell | 39     | 1037  | STATUS_ACTIVE | lp1       |
      | lp2   | ETH/DEC22 | buy  | 45     | 893   | STATUS_ACTIVE | lp2       |
      | lp2   | ETH/DEC22 | sell | 37     | 1107  | STATUS_ACTIVE | lp2       |
    And the liquidity fee factor should be "0.002" for the market "ETH/DEC22"
    # no fees in auction
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # observe that lp2 gets lower share of the fees despite the same commitment amount (that is due to their orders being much wider than those of lp1)
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 26     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 14     | ETH   |

    # modify lp2 orders so that they fall outside the price monitoring bounds
    When clear all events
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type   |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 | buy  | BID              | 1          | 116    | amendment |
      | lp2 | lp2   | ETH/DEC22 | 40000             | 0.002 | sell | ASK              | 1          | 116    |           |
    Then the orders should have the following states:
      | party | market id | side | volume | price | status        | reference |
      | lp1   | ETH/DEC22 | buy  | 42     | 963   | STATUS_ACTIVE | lp1       |
      | lp1   | ETH/DEC22 | sell | 39     | 1037  | STATUS_ACTIVE | lp1       |
      | lp2   | ETH/DEC22 | buy  | 46     | 879   | STATUS_ACTIVE | lp2       |
      | lp2   | ETH/DEC22 | sell | 36     | 1121  | STATUS_ACTIVE | lp2       |
    And the market data for the market "ETH/DEC22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 10000   | 893       | 1120      | 995                   | 1000             | 1005                    |
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # all the fees go to lp2
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 40     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 0      | ETH   |

    # lp2 manually adds some limit orders within PM range, observe automatically deployed orders go down and fee share go up
    When clear all events
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp2   | ETH/DEC22 | buy  | 15     | 995   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the orders should have the following states:
      | party | market id | side | volume | price | status        | reference |
      | lp1   | ETH/DEC22 | buy  | 42     | 963   | STATUS_ACTIVE | lp1       |
      | lp1   | ETH/DEC22 | sell | 39     | 1037  | STATUS_ACTIVE | lp1       |
      | lp2   | ETH/DEC22 | buy  | 29     | 879   | STATUS_ACTIVE | lp2       |
      | lp2   | ETH/DEC22 | sell | 36     | 1121  | STATUS_ACTIVE | lp2       |

    When clear all events
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | lp2   | ETH/DEC22 | sell | 15     | 1005  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the orders should have the following states:
      | party | market id | side | volume | price | status        | reference |
      | lp1   | ETH/DEC22 | buy  | 42     | 963   | STATUS_ACTIVE | lp1       |
      | lp1   | ETH/DEC22 | sell | 39     | 1037  | STATUS_ACTIVE | lp1       |
      | lp2   | ETH/DEC22 | buy  | 29     | 879   | STATUS_ACTIVE | lp2       |
      | lp2   | ETH/DEC22 | sell | 23     | 1121  | STATUS_ACTIVE | lp2       |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # lp2 has increased their liquidity score by placing limit orders closer to the mid (and within price monitoring bounds),
    # hence their fee share is larger (and no longer 0) now.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 29     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 11     | ETH   |

    # lp2 manually adds some pegged orders within PM range, liquidity obligation is now fullfiled by limit and pegged orders so no automatic order deployment takes place
    When clear all events
    And the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | lp2   | ETH/DEC22 | sell | 30     | ASK              | 1      |
    And the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | lp2   | ETH/DEC22 | buy  | 30     | BID              | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party2 | ETH/DEC22 | buy  | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/DEC22 | sell | 20     | 1000  | 1                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "40" for the market "ETH/DEC22"

    When the network moves ahead "6" blocks
    # lp2 has increased their liquidity score by placing pegged orders closer to the mid (and within price monitoring bounds),
    # hence their fee share is larger now. Since their liquidity score is higher than that of lp1 they get a higher payout.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 19     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC22 | 21     | ETH   |

  Scenario: 2 LPs joining at start, unequal commitments

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | MID              | 2          | 1      | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | MID              | 2          | 1      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | MID              | 2          | 1      | submission |

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
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 6    | lp1    |
      | party2 | 951   | 2    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"
    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 6    | lp1    |
      | party1 | 951   | 2    | lp2    |

    And the accumulated liquidity fees should be "8" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:20:06Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 6      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 2      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  Scenario: 2 LPs joining at start, unequal commitments, 1 LP joining later

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | lp3    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | MID              | 2          | 1      | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | ASK              | 1          | 2      | amendment  |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | MID              | 2          | 1      | amendment  |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | MID              | 2          | 1      | amendment  |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | ASK              | 1          | 2      | amendment  |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | MID              | 2          | 1      | amendment  |

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
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 6    | lp1    |
      | party2 | 951   | 2    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp3 | lp3   | ETH/DEC21 | 10000             | 0.001 | sell | MID              | 2          | 1      | submission |

    And the liquidity provider fee shares for the market "ETH/DEC21" should be:
      | party | equity like share | average entry valuation |
      | lp1   | 0.4               | 8000                    |
      | lp2   | 0.1               | 10000                   |
      | lp3   | 0.5               | 20000                   |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 16     | 1000  | 3                | TYPE_LIMIT | TIF_FOK |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 6    | lp1    |
      | party1 | 951   | 2    | lp2    |
      | party1 | 951   | 8    | lp3    |

    And the accumulated liquidity fees should be "16" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:20:06Z"
    # lp3 gets lower fee share than indicated by the ELS this fee round as it was later to deploy liquidity (so lower liquidity scores than others had)
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 12     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 3      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 1      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    # make sure we're in the next time window now
    When time is updated to "2019-11-30T00:30:07Z"
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/DEC21 | buy  | 16     | 1000  | 3                | TYPE_LIMIT | TIF_FOK |
    Then the accumulated liquidity fees should be "16" for the market "ETH/DEC21"

    When time is updated to "2019-11-30T00:40:08Z"
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 6      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 1      | ETH   |
      | market | lp3 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 9      | ETH   |
    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

  @Panic
  Scenario: 2 LPs joining at start, unequal commitments, market settles (0042-LIQF-014)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | MID              | 2          | 1      | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | MID              | 2          | 1      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | MID              | 2          | 1      | submission |

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
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 6    | lp1    |
      | party2 | 951   | 2    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"

    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |


    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 6    | lp1    |
      | party1 | 951   | 2    | lp2    |

    And the accumulated liquidity fees should be "8" for the market "ETH/DEC21"

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
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 6      | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 2      | ETH   |

  Scenario: 2 LPs joining at start, unequal commitments, 1 leaves later (0042-LIQF-012)

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | ETH   | 1000000000 |
      | lp2    | ETH   | 1000000000 |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | BID              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | buy  | MID              | 2          | 1      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | ASK              | 1          | 2      | submission |
      | lp1 | lp1   | ETH/DEC21 | 8000              | 0.001 | sell | MID              | 2          | 1      | submission |
    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | MID              | 2          | 1      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | ASK              | 1          | 2      | submission |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | sell | MID              | 2          | 1      | submission |

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
      | party2 | ETH/DEC21 | buy  | 20     | 1000  | 3                | TYPE_LIMIT | TIF_GTC | party2-buy  |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party2 | 951   | 6    | lp1    |
      | party2 | 951   | 2    | lp2    |
      | party2 | 1000  | 12   | party1 |

    And the accumulated liquidity fees should be "20" for the market "ETH/DEC21"
    # opening auction + time window
    Then time is updated to "2019-11-30T00:10:05Z"

    # these are different from the tests, but again, we end up with a 2/3 vs 1/3 fee share here.
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 16     | ETH   |
      | market | lp2 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 4      | ETH   |


    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/DEC21 | buy  | 40     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy  |
      | party2 | ETH/DEC21 | sell | 40     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-sell |

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 951   | 6    | lp1    |
      | party1 | 951   | 2    | lp2    |

    And the accumulated liquidity fees should be "8" for the market "ETH/DEC21"

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type      |
      | lp2 | lp2   | ETH/DEC21 | 2000              | 0.002 | buy  | BID              | 1          | 2      | cancellation |
    Then the liquidity provisions should have the following states:
      | id  | party | market    | commitment amount | status           |
      | lp2 | lp2   | ETH/DEC21 | 2000              | STATUS_CANCELLED |

    Then time is updated to "2019-11-30T00:20:06Z"

    # now all the accumulated fees go to the remaining lp (as the other one cancelled their provision)
    Then the following transfers should happen:
      | from   | to  | from account                | to account           | market id | amount | asset |
      | market | lp1 | ACCOUNT_TYPE_FEES_LIQUIDITY | ACCOUNT_TYPE_GENERAL | ETH/DEC21 | 8      | ETH   |

    And the accumulated liquidity fees should be "0" for the market "ETH/DEC21"
