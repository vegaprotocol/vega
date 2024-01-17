Feature: Test It is possible to configure a cash settled futures market to use a weighted average of 1. weighted average of trades over network.markPriceUpdateMaximumFrequency and 2. impact of leveraged notional on the order book with the value of USDT 100 and 3. an oracle source and if last trade is last updated more than 1 minute ago then it is removed and the remaining re-weighted and if the oracle is last updated more than 5 minutes ago then it is removed and the remaining re-weighted (0009-MRKP-018) and a perps market (with the oracle source different to that used for the external price in the perps market) (0009-MRKP-019).
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |

    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | price1.USD.value | TYPE_INTEGER | 0              |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0            | 1           | 100         | 1,4,5,0        | 1s,5s,20s,1h25m0s          | oracle1 |

  Scenario: 001 check mark price using weight average
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party            | ETH/FEB23 | sell | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15920 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15940 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    # leaving opening auction
    # mark price calcualted from the trade price
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 2      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15940 | 1                | TYPE_LIMIT | TIF_GTC |           |

    Then the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | price1.USD.value | 16000 | -1s         |

    When the network moves ahead "2" blocks
    # we have:
    # price from trades = 15930 - 2 trades in scope
    # price from book = 15900 - since the opening auction there are no orders on the sell side so not updating but still not stale
    # price from oracle = 16000
    # markprice = 0.1*15930 + 0.4*15900 + 0.5*16000
    Then the mark price should be "15953" for the market "ETH/FEB23"

    # now place and order and let trades go out of scope
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15940 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    # we have:
    # price from trades = 0 - no trades, price is already stale
    # price from book = 15930 fresh
    # price from oracle = 16000 still not stale
    # markprice = 4/9*15930 + 5/9*16000 = 15968
    Then the mark price should be "15968" for the market "ETH/FEB23"

    # trade the buy side to get rid of the book mid
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15920 | 1                | TYPE_LIMIT | TIF_GTC |           |

    # now we have no orders on the buy side
    When the network moves ahead "10" blocks
    # we have:
    # price from trades = 0 - no trades, price is already stale
    # price from book = 0 stale
    # price from oracle = 16000 still not stale
    # markprice = 16000 
    Then the mark price should be "16000" for the market "ETH/FEB23"

    # get the oracle price to stale and submit a fresh order
    When the network moves ahead "10" blocks
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15920 | 0                | TYPE_LIMIT | TIF_GTC |           |

    # now we only have a book price of 15930 (trade and oracle are stale) so it gets a weight of 1
    When the network moves ahead "2" blocks
    Then the mark price should be "15930" for the market "ETH/FEB23"