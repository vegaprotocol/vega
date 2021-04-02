Feature: Price monitoring test using simple risk model

  Background:
    Given the markets start on "2020-10-16T00:00:00Z" and expire on "2020-12-31T23:59:59Z"
    And the price monitoring updated every "60" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 240               |
      | 120     | 0.99        | 360               |
    And the simple risk model named "my-simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.11 | 0.1   | 10          | -11           | 0.1                    |
    And the markets:
      | id        | quote name | asset | auction duration | risk model           | margin calculator         | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 240              | my-simple-risk-model | default-margin-calculator | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 240                            |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount        |
      | trader1 | ETH   | 10000         |
      | trader2 | ETH   | 10000         |
      | aux     | ETH   | 100000000000  |
      | aux2    | ETH   | 100000000000  |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:14:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111"

  Scenario: GFN orders results in auction (issue #2657)
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    When traders place the following orders:
      | trader  | market id | side | volume | price | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      | 111   | TYPE_LIMIT | TIF_GFN | ref-1     |
    Then the system should return error "OrderError: invalid time in force"
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

  Scenario: Non-persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates.
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:14:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "100"

  Scenario: Non-persistent order results in an auction (both triggers breached), orders placed during auction result in a trade with indicative price within the price monitoring bounds, hence auction concludes.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 100    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      | 112   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:14:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111"

  Scenario: Persistent order results in an auction (one trigger breached), no orders placed during auction, auction gets extended due to 2nd trigger and eventually terminates with a trade from order that originally triggered the auction.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "110"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the initial auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:20:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:20:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "120"

  Scenario: Non-persistent order results in an auction (one trigger breached), no orders placed during auction and auction terminates

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "110"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

  Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in a trade with indicative price outside the price monitoring bounds, hence auction get extended, no further orders placed, auction concludes.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "110"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:20:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:20:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "133"

  Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in trade with indicative price outside the price monitoring bounds, hence auction get extended, additional orders resulting in more trades placed, auction concludes.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 10000        |
      | trader2 | ETH   | 10000        |
      | aux     | ETH   | 100000000000 |
      | aux2    | ETH   | 100000000000 |
  
    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 99     | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 134    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux2    | ETH/DEC20 | buy  | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 110    | 0                | TYPE_LIMIT  | TIF_GTC | 
    Then the opening auction period for market "ETH/DEC20" ends
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    And the mark price for the market "ETH/DEC20" is "110"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110"

    #T0 + 10s
    Then time is updated to "2020-10-16T00:08:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 115   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 115   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "115"

    #T0 + 01min10s
    Then time is updated to "2020-10-16T00:09:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 105   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 105   | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:10:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 120   | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 120   | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:14:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 2      | 133   | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:14:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:20:10Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 10     | 303   | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 10     | 303   | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "105"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:20:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "303"
