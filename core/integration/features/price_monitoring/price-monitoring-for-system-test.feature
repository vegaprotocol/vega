Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99845,100156], [99711,100290])

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 15      | 0.99        | 8                 |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 6                | default-none | my-price-monitoring | default-eth-for-future | 1e-4                   | 1e-4                      |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 6     |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Scenario for the system test with opening auction
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 100000000000 |
      | party2 | ETH   | 100000000000 |
      | party3 | ETH   | 100000000000 |
      | party4 | ETH   | 100000000000 |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 2      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC |

    And the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference     |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1         |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2         |
      | party3 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party3_buy_1  |
      | party4 | ETH/DEC20 | sell | 1      | 105000 | 0                | TYPE_LIMIT | TIF_GTC | party4_sell_1 |

    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    # T + 5s
    When time is updated to "2020-10-16T00:00:06Z"
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    # T + 1s
    When time is updated to "2020-10-16T00:00:07Z"
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 100000     | TRADING_MODE_CONTINUOUS | 5       | 99845     | 100156    |
      | 100000     | TRADING_MODE_CONTINUOUS | 15      | 99646     | 100355    |

    # We've left opening auction, cancel the orders we had to place on the book to allow for this to happen
    And the parties cancel the following orders:
      | party  | reference     |
      | party3 | party3_buy_1  |
      | party4 | party4_sell_1 |

    # 1st trigger breached with non-persistent order -> auction with initial duration of 6s starts
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party1 | ETH/DEC20 | sell | 1      | 99843 | 0                | TYPE_LIMIT | TIF_GTC | party1_sell_1 |
      | party2 | ETH/DEC20 | buy  | 1      | 99843 | 0                | TYPE_LIMIT | TIF_GTC | ref-3         |

    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the parties cancel the following orders:
      | party  | reference     |
      | party1 | party1_sell_1 |
      | party2 | ref-3         |

    # T + 4s
    When time is updated to "2020-10-16T00:00:10Z"

    # 2nd trigger breached with persistent order -> auction extended by 8s (total auction time no 14s).
    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100356 | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | party2 | ETH/DEC20 | buy  | 1      | 100356 | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |

    # T + 10s (last second of the auciton)
    Then time is updated to "2020-10-16T00:00:20Z"

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 1s (auction ended)
    Then time is updated to "2020-10-16T00:00:22Z"

    And the mark price should be "100356" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # 100291 is the new reference price, we get the following valid price ranges for the 2 triggers: [100135, 100447] & [100001, 100582]
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100447 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100447 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100356     | 100447            | TRADING_MODE_CONTINUOUS |


    # Now we should be after update and the bounds should change
    # T + 5s
    Then time is updated to "2020-10-16T00:00:26Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100448 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100448 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100356     | 100448            | TRADING_MODE_CONTINUOUS |


    # Now, we have the following valid price ranges for the 2 triggers: [100213, 100525] & [100079, 100660]
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party2 | ETH/DEC20 | buy  | 2      | 100213 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100050 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100356     | 100448            | TRADING_MODE_CONTINUOUS |


    # T + 2s
    When time is updated to "2020-10-16T00:00:28Z"

    # Both triggers breached with market order -> 14s auction
    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 3      | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |


    And the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 3s
    When time is updated to "2020-10-16T00:00:33Z"

    Then the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the parties cancel the following orders:
      | party  | reference |
      | party1 | ref-6     |
    # T + 8s
    When time is updated to "2020-10-16T00:00:43Z"

    Then the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # 100213 is the new reference price, we get the following valid price ranges for the 2 triggers: [100057, 100369] & [99923, 100503]

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100292 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100292 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100448     | 100292            | TRADING_MODE_CONTINUOUS |


    # T + 12s
    When time is updated to "2020-10-16T00:00:55Z"

    # Both triggers breached with persistent order -> auction with duration of 10s starts
    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100650 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100650 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |


    And the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 6s
    # T + 1s (min duration is 5 seconds, this test is broken)
    When time is updated to "2020-10-16T00:00:56Z"

    Then the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 1s (2 seconds)
    When time is updated to "2020-10-16T00:00:57Z"

    Then the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 8s (6s)
    When time is updated to "2020-10-16T00:01:12Z"

    Then the mark price should be "100650" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
