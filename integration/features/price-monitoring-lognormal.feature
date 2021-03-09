Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99460,100541], [98999,101008])

  Background:
    Given the markets start on "2020-10-16T00:00:00Z" and expire on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC20 | ETH        | ETH   | forward    | 0.000001  | 0.00011407711613050422 | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 0                | 0         | 0                  | 0             | 60                 | 60,120         | 0.95,0.99   | 240,360         | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100000"

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:10:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111000"


  Scenario: Non-persistent order results in an auction (both triggers breached), no orders placed during auction, auction terminates.
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100000"

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:10:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "100000"

  Scenario: Non-persistent order results in an auction (both triggers breached), orders placed during auction result in a trade with indicative price within the price monitoring bounds, hence auction concludes.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100000"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100000"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      | 112000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    #T0 + 10min
    Then time is updated to "2020-10-16T00:10:00Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "100000"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T00:10:01Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111500"

  Scenario: Persistent order results in an auction (one trigger breached), no orders placed during auction, auction terminates with a trade from order that originally triggered the auction.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 = T0 + 02min10s (auction start)
    Then time is updated to "2020-10-16T00:02:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:06:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s (auction ended)
    Then time is updated to "2020-10-16T00:06:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "111000"

  Scenario: Non-persistent order results in an auction (one trigger breached), no orders placed during auction and auction terminates

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 = T0 + 10s
    Then time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:04:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s
    Then time is updated to "2020-10-16T00:04:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110000"

  Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in a trade with indicative price outside the price monitoring bounds, hence auction get extended, no further orders placed, auction concludes.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 = T0 + 10s
    Then time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:04:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:04:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:10:10Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:10:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "133000"

  Scenario: Non-persistent order results in an auction (one trigger breached), orders placed during auction result in trade with indicative price outside the price monitoring bounds, hence auction get extended, additional orders resulting in more trades placed, auction concludes.

    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place following orders:
      | trader  | id        | type | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the market trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 = T0 + 10s
    Then time is updated to "2020-10-16T00:00:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 111000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 111000 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 04min00s (last second of the auction)
    Then time is updated to "2020-10-16T00:04:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 2      | 133000 | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    #T1 + 04min01s (auction extended due to 2nd trigger)
    Then time is updated to "2020-10-16T00:04:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 10min00s (last second of the extended auction)
    Then time is updated to "2020-10-16T00:10:10Z"

    Then traders place following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 10     | 303000 | 0                | TYPE_LIMIT | TIF_GFA | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 10     | 303000 | 0                | TYPE_LIMIT | TIF_GFA | ref-2     |

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    And the mark price for the market "ETH/DEC20" is "110000"

    #T1 + 10min01s (extended auction finished)
    Then time is updated to "2020-10-16T00:10:11Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    And the mark price for the market "ETH/DEC20" is "303000"
