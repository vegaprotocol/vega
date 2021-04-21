Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99460,100541], [98999,101008])

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring updated every "60" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 240               |
      | 120     | 0.99        | 360               |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | maturity date        | risk model                    | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | default-log-normal-risk-model | default-margin-calculator | 60               | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 60     |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10 min
    When time is updated to "2020-10-16T00:10:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10min 10s
    Then time is updated to "2020-10-16T00:10:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 11min01s
    Then time is updated to "2020-10-16T00:11:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "111000" for the market "ETH/DEC20"


  Scenario: Non-persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates.
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10 min
    When time is updated to "2020-10-16T00:10:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10min + 10s
    Then time is updated to "2020-10-16T00:10:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 11min01s
    Then time is updated to "2020-10-16T00:11:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

  Scenario: Non-persistent order results in an auction (both triggers breached), orders placed during auction result in a trade with indicative price within the price monitoring bounds, hence auction concludes.

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10 min
    When time is updated to "2020-10-16T00:10:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      | 112000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    #T0 + 10min + 1m (min auction duration)
    Then time is updated to "2020-10-16T00:11:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 11min01s (opening period, min auction duration + 1 second, auction is over)
    Then time is updated to "2020-10-16T00:11:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "111500" for the market "ETH/DEC20"

  Scenario: Persistent order results in an auction (one trigger breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110000" for the market "ETH/DEC20"

    #T0 + 10 min
    When time is updated to "2020-10-16T00:10:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:12:10Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:13:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s (auction ended)
    Then time is updated to "2020-10-16T00:13:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "111000" for the market "ETH/DEC20"

  Scenario: Non-persistent order results in an auction (one trigger breached), no orders placed during auction and auction terminates

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110000" for the market "ETH/DEC20"

    #T0 + 10 min
    When time is updated to "2020-10-16T00:10:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 = T0 + 10s
    Then time is updated to "2020-10-16T00:10:10Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:11:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s
    Then time is updated to "2020-10-16T00:11:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

  Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in a trade with indicative price outside the price monitoring bounds, hence auction get extended, no further orders placed, auction concludes.

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110000" for the market "ETH/DEC20"

    #T0 + 2 min (end of auction)
    When time is updated to "2020-10-16T00:02:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 = T0 + 10s
    Then time is updated to "2020-10-16T00:02:10Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:03:10Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:04:10Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:05:11Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:09:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "133000" for the market "ETH/DEC20"

  Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in trade with indicative price outside the price monitoring bounds, hence auction get extended, additional orders resulting in more trades placed, auction concludes.

    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     |
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux2    | ETH/DEC20 | buy  | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
      | aux     | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "110000" for the market "ETH/DEC20"

    #T0 + 2 min (end of auction)
    When time is updated to "2020-10-16T00:02:00Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 = T0 + 10s
    When time is updated to "2020-10-16T00:02:10Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 04min00s (last second of the auction)
    When time is updated to "2020-10-16T00:03:10Z"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    When time is updated to "2020-10-16T00:06:11Z"

    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 10min00s (last second of the extended auction)
    When time is updated to "2020-10-16T00:08:11Z"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 10     | 303000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 10     | 303000 | 0                | TYPE_LIMIT | TIF_GFA | ref-2-last|

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "110000" for the market "ETH/DEC20"

    #T1 + 10min01s (extended auction finished) // this is not finished, not order left in the book.
    Then time is updated to "2020-10-16T00:12:11Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "303000" for the market "ETH/DEC20"
