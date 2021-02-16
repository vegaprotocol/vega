Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99845,100156], [99711,100290])

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset |   markprice  | risk model |     lamd/long |              tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. |    p. m. horizons |  p. m. probs | p. m. durations | Prob of trading |
      | ETH/DEC20 | BTC      | ETH       | ETH   |      900000  | forward    |      0.000001 | 0.00011407711613050422 |              0 | 0.016           |   2.0 |            1.4 |            1.2 |           1.1 |              42 |           5 | continuous   |        0 |                 0 |            0 |                 4  |              5,10 |    0.95,0.99 |             6,8 | 0.1             |

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"

  Scenario: Scenario for the system test with opening auction
    Given the following traders:
      | name    |      amount  |
      | trader1 | 100000000000  |
      | trader2 | 100000000000  |
      | trader3 | 100000000000  |
      | trader4 | 100000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |

    Then traders place following orders with references:
      | trader  | id        | type | volume |    price  | resulting trades | type        | tif     | reference      |
      | trader3 | ETH/DEC20 | buy  |      1 |    80000  |                0 | TYPE_LIMIT  | TIF_GTC | trader3_buy_1  |
      | trader4 | ETH/DEC20 | sell |      1 |   105000  |                0 | TYPE_LIMIT  | TIF_GTC | trader4_sell_1 |

    And the mark price for the market "ETH/DEC20" is "900000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"

    # T + 5s
    Then the time is updated to "2020-10-16T00:00:05Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"

    # T + 1s
    Then the time is updated to "2020-10-16T00:00:06Z"

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # We've left opening auction, cancel the orders we had to place on the book to allow for this to happen
    Then traders cancels the following orders reference:
      | trader   |       reference |
      | trader3  |   trader3_buy_1 |
      | trader4  |  trader4_sell_1 |

    # 1st trigger breached with non-persistent order -> auction with initial duration of 6s starts
    Then traders place following orders with references:
      | trader  | id        | type | volume |    price  | resulting trades | type        | tif     | reference      |
      | trader1 | ETH/DEC20 | sell |      1 |    99844  |                0 | TYPE_LIMIT  | TIF_GTC | trader1_sell_1 |

    Then traders place following orders:
      | trader  | id        | type | volume |    price | resulting trades | type       | tif     |
      | trader2 | ETH/DEC20 | buy  |      1 |    99844 |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    Then traders cancels the following orders reference:
      | trader   |       reference |
      | trader1  |  trader1_sell_1 |

    # T + 4s
    Then the time is updated to "2020-10-16T00:00:10Z"

  Then dump orders

    # 2nd trigger breached with persistent order -> auction extended by 8s (total auction time no 14s).
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100291  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100291  |                0 | TYPE_LIMIT | TIF_GTC |

    # T + 10s (last second of the auciton)
    Then the time is updated to "2020-10-16T00:00:20Z"

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 1s (auction ended)
    Then the time is updated to "2020-10-16T00:00:21Z"

    And the mark price for the market "ETH/DEC20" is "100291"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And dump transfers

    # 100291 is the new reference price, we get the following valid price ranges for the 2 triggers: [100135, 100447] & [100001, 100582]
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100447  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100447  |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC20" is "100447"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # Now we should be after update and the bounds should change
    # T + 5s
    Then the time is updated to "2020-10-16T00:00:26Z"

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100448  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100448  |                1 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC20" is "100448"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # Now, we have the following valid price ranges for the 2 triggers: [100213, 100525] & [100079, 100660]
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader2 | ETH/DEC20 | buy  |      2 |   100213  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100050  |                0 | TYPE_LIMIT | TIF_GTC |

    And the mark price for the market "ETH/DEC20" is "100448"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"


    # T + 2s
    Then the time is updated to "2020-10-16T00:00:28Z"

    # Both triggers breached with market order -> 14s auction
    Then traders place following orders:
      | trader  | id        | type  | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell  |      3 |        0  |                0 | TYPE_MARKET | TIF_FOK |


    And the mark price for the market "ETH/DEC20" is "100448"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 7s
    Then the time is updated to "2020-10-16T00:00:35Z"

    And the mark price for the market "ETH/DEC20" is "100448"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 8s
    Then the time is updated to "2020-10-16T00:00:43Z"

    And the mark price for the market "ETH/DEC20" is "100448"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # 100213 is the new reference price, we get the following valid price ranges for the 2 triggers: [100057, 100369] & [99923, 100503]

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100292  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100292  |                1 | TYPE_LIMIT | TIF_GTC |


    And the mark price for the market "ETH/DEC20" is "100292"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"


    # T + 12s
    Then the time is updated to "2020-10-16T00:00:55Z"

     # Both triggers breached with persistent order -> auction with duration of 10s starts
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100650  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100650  |                0 | TYPE_LIMIT | TIF_GTC |


    And the mark price for the market "ETH/DEC20" is "100292"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 6s
    Then the time is updated to "2020-10-16T00:01:01Z"

    And the mark price for the market "ETH/DEC20" is "100292"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 1s
    Then the time is updated to "2020-10-16T00:01:02Z"

    And the mark price for the market "ETH/DEC20" is "100292"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 8s
    Then the time is updated to "2020-10-16T00:01:10Z"

    And the mark price for the market "ETH/DEC20" is "100650"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
