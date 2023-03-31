Feature: Price monitoring test for issue 2668

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 43200   | 0.9999999   | 300               |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 0.8   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 1e-4                    | 1e-4                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 300   |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Upper bound breached
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount       |
      | party1    | ETH   | 10000000000  |
      | party2    | ETH   | 10000000000  |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |
      | lpprov    | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 5670000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 5670000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "5670000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 4850000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 4850000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "4850000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 6630000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 6630000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 6640000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 6640000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # enter price monitoring auction
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC20"
    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 10min (5 min opening auction + 5 min price monitoring)
    Then time is updated to "2020-10-16T00:10:00Z"

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 15min01s (opening auction, price monitoring + extension due to time update + another period)
    When time is updated to "2020-10-16T00:15:01Z"
    # leave auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC20"

    # the order was GTC, so after the auction this trade can now happen
    And the mark price should be "6640000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

  Scenario: Lower bound breached
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount       |
      | party1    | ETH   | 10000000000  |
      | party2    | ETH   | 10000000000  |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |
      | lpprov    | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party     | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 5670000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 5670000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 5670000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "5670000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 4850000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 4850000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "4850000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 6630000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 6630000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 4840000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 4840000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    #price monitoring auction started
    Then the market state should be "STATE_SUSPENDED" for the market "ETH/DEC20"
    And the mark price should be "6630000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the mark price should be "6630000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 15min01s
    When time is updated to "2020-10-16T00:15:01Z"
    # leave auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC20"
    And the mark price should be "4840000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

  Scenario: Upper bound breached (scale prices down by 10000)
    Given the parties deposit on asset's general account the following amount:
      | party     | asset | amount       |
      | party1    | ETH   | 10000000000  |
      | party2    | ETH   | 10000000000  |
      | auxiliary | ETH   | 100000000000 |
      | aux2      | ETH   | 100000000000 |
      | lpprov    | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party     | market id | side | volume | price    | resulting trades | type       | tif     |
      | auxiliary | ETH/DEC20 | buy  | 1      | 1        | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 10000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | auxiliary | ETH/DEC20 | sell | 1      | 567      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2      | ETH/DEC20 | buy  | 1      | 567      | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 567   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 567   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    Then the mark price should be "567" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 485   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 485   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    Then the mark price should be "485" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 663   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 663   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    Then the mark price should be "663" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 665   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 665   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the mark price should be "663" for the market "ETH/DEC20"
    # enter price monitoring auction
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the market state should be "STATE_SUSPENDED" for the market "ETH/DEC20"

    # T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the mark price should be "663" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T0 + 15min01s
    When time is updated to "2020-10-16T00:15:01Z"
    # leave auction
    Then the market state should be "STATE_ACTIVE" for the market "ETH/DEC20"
    And the mark price should be "665" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
