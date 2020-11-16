Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99845,100156], [99711,100290])

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the executon engine have these markets:
      | name      | baseName | quoteName | asset |   markprice  | risk model |     lamd/long |              tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq. |    p. m. horizons |  p. m. probs | p. m. durations |
      | ETH/DEC20 | BTC      | ETH       | ETH   |      900000  | forward    |      0.000001 | 0.00011407711613050422 |              0 | 0.016           |   2.0 |            1.4 |            1.2 |           1.1 |              42 |           0 | continuous   |        0 |                 0 |            0 |                 4  |              5,10 |    0.95,0.99 |             6,8 |

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

  Scenario: Scenario for the system test, ideally it should include the opening auction where the price of 100000 will get established.
    Given the following traders:
      | name    |      amount  |
      | trader1 | 100000000000  |
      | trader2 | 100000000000  |

  # WG: Ideally, this should come from the opening auction
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    # T + 1s
    Then the time is updated to "2020-10-16T00:00:01Z" 

    # 1st trigger breached with non-persistent order -> auction with initial duration of 6s starts
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   99844  |                0 | TYPE_LIMIT  | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   99844  |                0 | TYPE_LIMIT  | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    # T + 4s
    Then the time is updated to "2020-10-16T00:00:05Z" 

    # 2nd trigger breached with persistent order -> auction extended by 8s (total auction time no 14s).
    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100291  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100291  |                0 | TYPE_LIMIT | TIF_GTC |

    # T + 10s (last second of the auciton)
    Then the time is updated to "2020-10-16T00:00:15Z" 

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_MONITORING_AUCTION"

    # T + 1s (auction ended)
    Then the time is updated to "2020-10-16T00:00:16Z" 

    And the mark price for the market "ETH/DEC20" is "100291"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    # 100291 is the new reference price, we get the following valid price ranges for the 2 triggers: [100135, 100447] & [100001, 100582]

  Then dump orders

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100391  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100391  |                0 | TYPE_LIMIT | TIF_GTC |

    Then the time is updated to "2020-10-16T00:00:20Z" 

    Then dump orders

    And the mark price for the market "ETH/DEC20" is "100291"

    And the market state for the market "ETH/DEC20" is "MARKET_STATE_CONTINUOUS"

    # T + 5s (bounds should get updated)
    #Then the time is updated to "2020-10-16T00:00:16Z" 
