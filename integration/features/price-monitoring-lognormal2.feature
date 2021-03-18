Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [95878,104251], [90497,110401])

  Background:
    Given the markets start on "2020-10-16T00:00:00Z" and expire on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC20 | ETH        | ETH   | forward    | 0.000001  | 0.00011407711613050422 | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 1                | 0         | 0                  | 0             | 6000               | 3600,7200      | 0.95,0.999  | 240,360         | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Auction triggered by 1st trigger (lower bound breached)
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |

    # Trigger an auction to set the mark price
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period for market "ETH/DEC20" ends
    And the mark price for the market "ETH/DEC20" is "100000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95877 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T00:04:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T00:04:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by 1st trigger, upper bound
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |

    # Trigger an auction to set the mark price
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period for market "ETH/DEC20" ends
    And the mark price for the market "ETH/DEC20" is "100000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T00:04:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T00:04:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by 1 trigger (upper bound breached)
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |

    # Trigger an auction to set the mark price
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period for market "ETH/DEC20" ends
    And the mark price for the market "ETH/DEC20" is "100000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T00:04:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T00:04:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by both triggers (lower bound breached)
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |

    # Trigger an auction to set the mark price
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period for market "ETH/DEC20" ends
    And the mark price for the market "ETH/DEC20" is "100000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 90496 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 90496 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T00:04:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T00:04:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min + 2 second opening auction
    Then time is updated to "2020-10-16T00:10:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min03s
    Then time is updated to "2020-10-16T00:10:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by both triggers, upper bound
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |

    # Trigger an auction to set the mark price
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period for market "ETH/DEC20" ends
    And the mark price for the market "ETH/DEC20" is "100000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110402 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110402 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T00:04:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T00:04:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min + 2 second opening auction
    Then time is updated to "2020-10-16T00:10:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min03s
    Then time is updated to "2020-10-16T00:10:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "104251"

  Scenario: Auction triggered by 1st trigger (lower bound breached), extended by second (upper bound)
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |

    # Trigger an auction to set the mark price
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period for market "ETH/DEC20" ends
    And the mark price for the market "ETH/DEC20" is "100000"
    Then traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    Then time is updated to "2020-10-16T00:00:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "95878"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | cancel-me |
      | trader2 | ETH/DEC20 | buy  | 1      | 95877 | 0                | TYPE_LIMIT | TIF_FOK |           |

    And the mark price for the market "ETH/DEC20" is "104251"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 4min
    Then time is updated to "2020-10-16T00:04:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    Then traders cancel the following orders:
      | trader  | reference |
      | trader1 | cancel-me |

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110430 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110430 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    #T0 + 4min01s
    Then time is updated to "2020-10-16T00:04:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:02Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "104251"

    #T0 + 10min01sec
    Then time is updated to "2020-10-16T00:10:03Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110430"
