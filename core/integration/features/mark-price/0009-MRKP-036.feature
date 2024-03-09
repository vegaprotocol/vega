Feature: Test It is possible to configure a cash settled futures and perps market to use median
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 4s    |
    And the perpetual oracles from "0xCAFECAFE1":
      | name        | asset | settlement property | settlement type | schedule property | schedule type  | margin funding factor | interest rate | clamp lower bound | clamp upper bound | quote name | settlement decimals |
      | perp-oracle | USD   | perp.ETH.value      | TYPE_INTEGER    | perp.funding.cue  | TYPE_TIMESTAMP | 0.5                   | 0.05          | 0.1               | 0.9               | ETH        | 18                  |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 36000   | 0.95        | 6                 |

    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | price1.USD.value | TYPE_INTEGER | 0              |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 | market type |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures | median     | 0            | 1           | 20000       | 1,0,0,0        | 5s,20s,20s,1h25m0s         | oracle1 | future      |
      | ETH/FEB24 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | perp-oracle            | 0.25                   | 0                         | default-futures | median     | 0            | 1           | 20000       | 1,0,0,0        | 5s,20s,20s,1h25m0s         | oracle1 | perp        |

  Scenario: 001 check mark price using median with traded mark price and book mark price, 0009-MRKP-036, 0009-MRKP-037
    Given the parties deposit on asset's general account the following amount:
      | party             | asset | amount       |
      | buySideProvider   | USD   | 100000000000 |
      | sellSideProvider  | USD   | 100000000000 |
      | party             | USD   | 48050        |
      | buySideProvider1  | USD   | 100000000000 |
      | sellSideProvider1 | USD   | 100000000000 |
      | party1            | USD   | 48050        |
    And the parties place the following orders:
      | party             | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider   | ETH/FEB23 | buy  | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party             | ETH/FEB23 | sell | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 15920 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider  | ETH/FEB23 | sell | 1      | 15990 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider  | ETH/FEB23 | sell | 2      | 16008 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1  | ETH/FEB24 | buy  | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1            | ETH/FEB24 | sell | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider1 | ETH/FEB24 | sell | 1      | 15920 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider1 | ETH/FEB24 | sell | 1      | 15990 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider1 | ETH/FEB24 | sell | 2      | 16008 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    # leaving opening auction
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15900      | TRADING_MODE_CONTINUOUS | 36000   | 15801     | 15999     | 0            | 0              | 3             |
    Then the market data for the market "ETH/FEB24" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15900      | TRADING_MODE_CONTINUOUS | 36000   | 15801     | 15999     | 0            | 0              | 3             |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 2      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1 | ETH/FEB24 | buy  | 2      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "5" blocks

    # price from trades = 15920
    # price from book = 15900
    # markprice = median(15920,15900)=15910
    Then the mark price should be "15910" for the market "ETH/FEB23"
    Then the mark price should be "15910" for the market "ETH/FEB24"
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15990 | 1                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1 | ETH/FEB24 | buy  | 1      | 15990 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "1" blocks
    Then the mark price should be "15910" for the market "ETH/FEB23"
    Then the mark price should be "15910" for the market "ETH/FEB24"

    #0032-PRIM-039:For all available mark price calculation methodologies: the price history used by the price monitoring engine is in line with market's mark price history.
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | ref price | min bound | max bound | target stake | supplied stake | open interest |
      | 15910      | TRADING_MODE_CONTINUOUS | 36000   | 15900     | 15801     | 15999     | 0            | 0              | 5             |
    Then the market data for the market "ETH/FEB24" should be:
      | mark price | trading mode            | horizon | ref price | min bound | max bound | target stake | supplied stake | open interest |
      | 15910      | TRADING_MODE_CONTINUOUS | 36000   | 15900     | 15801     | 15999     | 0            | 0              | 5             |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 16008 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider1 | ETH/FEB24 | buy  | 1      | 16008 | 0                | TYPE_LIMIT | TIF_GTC |           |

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15910      | TRADING_MODE_MONITORING_AUCTION |

    When the network moves ahead "6" blocks
    Then the mark price should be "15910" for the market "ETH/FEB23"
    Then the mark price should be "15910" for the market "ETH/FEB24"

    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15910      | TRADING_MODE_MONITORING_AUCTION |

    Then the market data for the market "ETH/FEB24" should be:
      | mark price | trading mode                    |
      | 15910      | TRADING_MODE_MONITORING_AUCTION |

    # price from trades = (16008+15990)/2=15999
    # price from book = 16008
    # markprice = median(15999,16008)=16003
    When the network moves ahead "1" blocks
    Then the mark price should be "16003" for the market "ETH/FEB23"
    Then the mark price should be "16003" for the market "ETH/FEB24"
    #0009-MRKP-036:When a futures market is in a monitoring auction, book price is undefined with staleness increasing with time, the book price at auction uncrossing should be set to the price of the uncrossing trade, the mark price should only be recalculated when the auction exits, starting from only the last period indicated by `network.markPriceUpdateMaximumFrequency`
    #0032-PRIM-039
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | ref price | min bound | max bound | target stake | supplied stake | open interest |
      | 16003      | TRADING_MODE_CONTINUOUS | 36000   | 0         | 15909     | 16107     | 0            | 0              | 6             |
    #0009-MRKP-037
    Then the market data for the market "ETH/FEB24" should be:
      | mark price | trading mode            | horizon | ref price | min bound | max bound | target stake | supplied stake | open interest |
      | 16003      | TRADING_MODE_CONTINUOUS | 36000   | 0         | 15909     | 16107     | 0            | 0              | 6             |


