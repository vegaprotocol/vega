Feature: check pegged GTT and GTC in auction

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.55690359157934000
    #risk factor long = 0.801225765
    And the margin calculator named "margin-calculator-0":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 36600   | 0.99        | 300               |
    And the following network parameters are set:
      | name                              | value |
      | market.auction.minimumDuration    | 1     |
      | market.stake.target.scalingFactor | 1     |
      | limits.markets.maxPeggedOrders    | 1500  |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC19 | ETH        | ETH   | log-normal-risk-model-1 | margin-calculator-0 | 1                | default-none | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |

  Scenario: 001, Pegged GTC (good till time) (parked in auction), Pegged orders will be [parked] if placed during [an auction], with time priority preserved. 0011-MARA-017
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    # submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | submission |

    # get out of auction
    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | auxiliary | ETH/DEC19 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTC | oa-b-1    |
      | auxiliary | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-1    |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-b-2    |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | oa-s-2    |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100        | TRADING_MODE_CONTINUOUS | 36600   | 92        | 109       | 355          | 3000           | 1             |

    # add a few pegged orders now
    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2  | ETH/DEC19 | sell | 10     | ASK              | 2      |
      | aux2  | ETH/DEC19 | buy  | 5      | BID              | 4      |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 130   | 24     |
      | sell | 122   | 10     |
      | sell | 120   | 20     |
      | buy  | 80    | 20     |
      | buy  | 76    | 5      |
      | buy  | 70    | 43     |

    # now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |
    And the network moves ahead "1" blocks

    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 122   | 0      |
      | sell | 120   | 20     |
      | buy  | 80    | 20     |
      | buy  | 76    | 0      |


  Scenario: 002, Pegged GTT (good till time) (parked in auction), Pegged orders will be [parked] if placed during [an auction], with time priority preserved. 0011-MARA-016
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount    |
      | party1           | ETH   | 100000000 |
      | sellSideProvider | ETH   | 100000000 |
      | buySideProvider  | ETH   | 100000000 |
      | auxiliary        | ETH   | 100000000 |
      | aux2             | ETH   | 100000000 |

    # submit our LP
    Then the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | buy  | BID              | 50         | 10     | submission |
      | lp1 | party1 | ETH/DEC19 | 3000              | 0.1 | sell | ASK              | 50         | 10     | submission |

    # get out of auction
    When the parties place the following orders:
      | party     | market id | side | volume | price | resulting trades | type       | tif     | reference | expires in |
      | auxiliary | ETH/DEC19 | buy  | 20     | 80    | 0                | TYPE_LIMIT | TIF_GTT | oa-b-1    | 6          |
      | auxiliary | ETH/DEC19 | sell | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTT | oa-s-1    | 6          |
      | aux2      | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTT | oa-b-2    | 6          |
      | auxiliary | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTT | oa-s-2    | 6          |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"

    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 100        | TRADING_MODE_CONTINUOUS | 36600   | 92        | 109       | 355          | 3000           | 1             |

    # add a few pegged orders now
    Then the parties place the following pegged orders:
      | party | market id | side | volume | pegged reference | offset |
      | aux2  | ETH/DEC19 | sell | 10     | ASK              | 2      |
      | aux2  | ETH/DEC19 | buy  | 5      | BID              | 4      |

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 130   | 24     |
      | sell | 122   | 10     |
      | sell | 120   | 20     |
      | buy  | 80    | 20     |
      | buy  | 76    | 5      |
      | buy  | 70    | 43     |

    # now consume all the volume on the sell side
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 20     | 120   | 0                | TYPE_LIMIT | TIF_GTC | t1-1      |
    And the network moves ahead "1" blocks

    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"

    Then the order book should have the following volumes for market "ETH/DEC19":
      | side | price | volume |
      | sell | 122   | 0      |
      | sell | 120   | 20     |
      | buy  | 80    | 20     |
      | buy  | 76    | 0      |


