Feature: Price monitoring test using simple risk model

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 240               |
      | 600     | 0.99        | 360               |
    And the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.11 | 0.1   | 10          | 11            | 0.1                    |
    And the markets:
      | id        | quote name | asset | auction duration | risk model           | margin calculator         | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | 240              | my-simple-risk-model | default-margin-calculator | default-none | my-price-monitoring | default-eth-for-future | 0.01                   | 0                         |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 240   |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100" for the market "ETH/DEC20"
    And time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100" for the market "ETH/DEC20"

    #T0 + 10min
    Then time is updated to "2020-10-16T00:20:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:20:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "111" for the market "ETH/DEC20"

  Scenario: GFN orders don't result in auction
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    When the parties place the following orders:
      | party  | market id | side | volume | price | type       | tif     | reference | error                                                       |
      | party2 | ETH/DEC20 | buy  | 1      | 111   | TYPE_LIMIT | TIF_GFN | ref-1     | OrderError: non-persistent order trades out of price bounds |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

  Scenario: Non-persistent order results in an auction (both triggers breached), orders placed during auction result in a trade with indicative price within the price monitoring bounds, hence auction concludes.

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100" for the market "ETH/DEC20"
    Then time is updated to "2020-10-16T00:10:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC20 | buy  | 1      | 112   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    #T0 + 10min
    Then time is updated to "2020-10-16T00:20:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100" for the market "ETH/DEC20"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:20:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "111" for the market "ETH/DEC20"

  Scenario: Persistent order results in an auction (one trigger breached), no orders placed during auction, auction gets extended due to 2nd trigger and eventually terminates with a trade from order that originally triggered the auction.

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110" for the market "ETH/DEC20"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 110        | 115               | TRADING_MODE_CONTINUOUS |


    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 115        | 105               | TRADING_MODE_CONTINUOUS |

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the initial auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:20:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:20:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "120" for the market "ETH/DEC20"

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction and auction terminates

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |

    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux   | ETH/DEC20 | 660               | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux   | ETH/DEC20 | 660               | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110" for the market "ETH/DEC20"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 110        | 115               | TRADING_MODE_CONTINUOUS |

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "115" for the market "ETH/DEC20"

    #T1 = T0 + 04min10s (auction start)
    Then time is updated to "2020-10-16T00:12:10Z"

    And the market data for the market "ETH/DEC20" should be:
      | horizon | min bound | max bound |
      | 60      | 95        | 114       |
      | 600     | 99        | 119       |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | extension trigger           | target stake | supplied stake | auction end | horizon | min bound | max bound |
      | 105        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_UNSPECIFIED | 660          | 660            | 240         | 600     | 99        | 119       |

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of initial auction duration)
    When time is updated to "2020-10-16T00:16:10Z"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | extension trigger           | target stake | supplied stake | auction end | horizon | min bound | max bound |
      | 105        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_UNSPECIFIED | 660          | 660            | 240         | 600     | 99        | 119       |

    #T1 + 04min01s
    When time is updated to "2020-10-16T00:16:11Z"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | extension trigger           | target stake | supplied stake | auction end |
      | 105        | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_PRICE       | 660          | 660            | 600         |

    #T1 + 10min00s (last second of the extended auction)
    When time is updated to "2020-10-16T00:22:10Z"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    
    #T1 + 10min01s
    Then time is updated to "2020-10-16T00:22:11Z"
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | target stake | supplied stake |
      | 120        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 660          | 660            |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "120" for the market "ETH/DEC20"

  Scenario: Persistent order results in an auction (one trigger breached), orders placed during auction result in a trade with indicative price outside the price monitoring bounds, hence auction get extended, no further orders placed, auction concludes.

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110" for the market "ETH/DEC20"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 110        | 115               | TRADING_MODE_CONTINUOUS |

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 115        | 105               | TRADING_MODE_CONTINUOUS |

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | party2 | ETH/DEC20 | buy  | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:20:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:20:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "133" for the market "ETH/DEC20"

  Scenario: Persistent order results in an auction (one trigger breached), orders placed during auction result in trade with indicative price outside the price monitoring bounds, hence auction get extended, additional orders resulting in more trades placed, auction concludes.

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000        |
      | party2 | ETH   | 10000        |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 134   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110" for the market "ETH/DEC20"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 110        | 115               | TRADING_MODE_CONTINUOUS |

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 115        | 105               | TRADING_MODE_CONTINUOUS |

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | party2 | ETH/DEC20 | buy  | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:20:10Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 10     | 303   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 10     | 303   | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "105" for the market "ETH/DEC20"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:20:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "303" for the market "ETH/DEC20"
