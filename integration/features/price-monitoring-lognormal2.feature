Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [95878,104251], [90497,110401])

  Background:
    Given the markets starts on "2020-10-16T00:00:00Z" and expires on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | baseName | quoteName | asset |   markprice  | risk model |     lamd/long |              tau/short | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | settlementPrice | openAuction | trading mode | makerFee | infrastructureFee | liquidityFee | p. m. update freq.   |    p. m. horizons | p. m. probs  | p. m. durations | Prob of trading |
      | ETH/DEC20 | BTC      | ETH       | ETH   |      100000  | forward    |      0.000001 | 0.00011407711613050422 |              0 | 0.016           |   2.0 |            1.4 |            1.2 |           1.1 |              42 |           0 | continuous   |        0 |                 0 |            0 | 6000                 |         3600,7200 |   0.95,0.999 |         240,360 | 0.1             |

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

  Scenario: Auction triggered by 1st trigger (lower bound breached)
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104251  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104251  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

      Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95877  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95877  |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then the time is updated to "2020-10-16T00:04:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min01s
    Then the time is updated to "2020-10-16T00:04:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

   Scenario: Auction triggered by 1st trigger, upper bound
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104251  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104251  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104252  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104252  |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then the time is updated to "2020-10-16T00:04:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min01s
    Then the time is updated to "2020-10-16T00:04:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by 1 trigger (upper bound breached)
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104251  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104251  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

      Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104252  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104252  |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then the time is updated to "2020-10-16T00:04:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min01s
    Then the time is updated to "2020-10-16T00:04:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by both triggers (lower bound breached)
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104251  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104251  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

      Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   90496  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   90496  |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then the time is updated to "2020-10-16T00:04:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min01s
    Then the time is updated to "2020-10-16T00:04:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

    Scenario: Auction triggered by both triggers, upper bound
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104251  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104251  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
    | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   110402  |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   110402  |                0 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then the time is updated to "2020-10-16T00:04:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min01s
    Then the time is updated to "2020-10-16T00:04:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:00Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by 1st trigger (lower bound breached), extended by second (upper bound)
    Given the following traders:
      | name    |      amount  |
      | trader1 | 10000000000  |
      | trader2 | 10000000000  |

    Then traders place following orders:
      | trader  | id        | type | volume |    price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   100000  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   100000  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    Then the time is updated to "2020-10-16T00:00:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   95878  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   95878  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | id        | type | volume |   price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell |      1 |   104251  |                0 | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  |      1 |   104251  |                1 | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders with references:
    | trader  | id        | type | volume |   price  | resulting trades | type       | tif     | reference |
    | trader1 | ETH/DEC20 | sell |      1 |   95877  |                0 | TYPE_LIMIT | TIF_GTC | cancel-me |
    | trader2 | ETH/DEC20 | buy  |      1 |   95877  |                0 | TYPE_LIMIT | TIF_FOK |           |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then the time is updated to "2020-10-16T00:04:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

  Then traders cancels the following orders reference:
    | trader   |  reference |
    | trader1  |  cancel-me |

  Then traders place following orders:
    | trader  | id        | type | volume |   price   | resulting trades | type       | tif     |
    | trader1 | ETH/DEC20 | sell |      1 |   110402  |                0 | TYPE_LIMIT | TIF_GTC |
    | trader2 | ETH/DEC20 | buy  |      1 |   110402  |                0 | TYPE_LIMIT | TIF_GTC |

    #T0 + 4min01s
    Then the time is updated to "2020-10-16T00:04:02Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min
    Then the time is updated to "2020-10-16T00:10:01Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min01sec
    Then the time is updated to "2020-10-16T00:10:02Z"

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110402"
