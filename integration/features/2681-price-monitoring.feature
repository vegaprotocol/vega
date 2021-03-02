Feature: Price monitoring test for issue 2681

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset |   markprice  | risk model |         lamda  |              tau/short  | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq.   |    p. m. horizons | p. m. probs  | p. m. durations | Prob of trading |
      | ETH/DEC20 | BTC      | ETH       | ETH   |      5000000  | forward   |       0.000001 |  0.00011407711613050422 |              0 | 0.016           |   0.8 |            1.4 |            1.2 |           1.1 |              42 |           0 | continuous   |        0 |                 0 |            0 | 1                    |             43200 |    0.9999999 |             300 |                 |

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

  Scenario: Upper bound breached
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price   | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   5670000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   5670000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "5670000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # T0 + 1min - this causes the price for comparison of the bounds to be 567
    Then the time is updated to "2020-10-16T00:01:00Z"

    Then traders place following orders:
      | trader  | id        | type | volume |   price    | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   4850000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   4850000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "4850000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # T0 + 2min
    Then the time is updated to "2020-10-16T00:02:00Z"

    Then traders place following orders:
      | trader  | id        | type | volume |   price    | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   6490000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   6490000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "6490000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # T0 + 3min
    # The reference price is still 5670000
    Then the time is updated to "2020-10-16T00:03:00Z"

    Then traders place following orders:
      | trader  | id        | type | volume |   price    | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   6635392    |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   6635392  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "6635392"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price    | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   6635393  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   6635393  |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "6635392"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"
