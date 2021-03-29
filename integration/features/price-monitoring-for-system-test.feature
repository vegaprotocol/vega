Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99845,100156], [99711,100290])

  Background:
    Given the markets start on "2020-10-16T00:00:00Z" and expire on "2020-12-31T23:59:59Z"
    And the execution engine have these markets:
      | name      | quote name | asset | risk model | lamd/long | tau/short              | mu/max move up | r/min move down | sigma | release factor | initial factor | search factor | auction duration | maker fee | infrastructure fee | liquidity fee | p. m. update freq. | p. m. horizons | p. m. probs | p. m. durations | prob. of trading | oracle spec pub. keys | oracle spec property | oracle spec property type | oracle spec binding |
      | ETH/DEC20 | ETH        | ETH   | forward    | 0.000001  | 0.00011407711613050422 | 0              | 0.016           | 2.0   | 1.4            | 1.2            | 1.1           | 5                | 0         | 0                  | 0             | 4                  | 5,10           | 0.95,0.99   | 6,8             | 0.1              | 0xDEADBEEF,0xCAFEDOOD | prices.ETH.value     | TYPE_INTEGER              | prices.ETH.value    |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 5                              |
    And oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"

  Scenario: Scenario for the system test with opening auction
    Given the traders make the following deposits on asset's general account:
      | trader  | asset | amount       |
      | trader1 | ETH   | 100000000000 |
      | trader2 | ETH   | 100000000000 |
      | trader3 | ETH   | 100000000000 |
      | trader4 | ETH   | 100000000000 |
      | aux     | ETH   | 100000000000 |
      
     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference      |
      | trader3 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader3_buy_1  |
      | trader4 | ETH/DEC20 | sell | 1      | 105000 | 0                | TYPE_LIMIT | TIF_GTC | trader4_sell_1 |

    And the mark price for the market "ETH/DEC20" is "0"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"

    # T + 5s
    Then time is updated to "2020-10-16T00:00:05Z"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_OPENING_AUCTION"

    # T + 1s
    Then time is updated to "2020-10-16T00:00:06Z"

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # We've left opening auction, cancel the orders we had to place on the book to allow for this to happen
    Then traders cancel the following orders:
      | trader  | reference      |
      | trader3 | trader3_buy_1  |
      | trader4 | trader4_sell_1 |

    # 1st trigger breached with non-persistent order -> auction with initial duration of 6s starts
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader1 | ETH/DEC20 | sell | 1      | 99844 | 0                | TYPE_LIMIT | TIF_GTC | trader1_sell_1 |

    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 1      | 99844 | 0                | TYPE_LIMIT | TIF_FOK | ref-1     |

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    Then traders cancel the following orders:
      | trader  | reference      |
      | trader1 | trader1_sell_1 |

    # T + 4s
    Then time is updated to "2020-10-16T00:00:10Z"

    # 2nd trigger breached with persistent order -> auction extended by 8s (total auction time no 14s).
    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    # T + 10s (last second of the auciton)
    Then time is updated to "2020-10-16T00:00:20Z"

    And the mark price for the market "ETH/DEC20" is "100000"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 1s (auction ended)
    Then time is updated to "2020-10-16T00:00:21Z"

    And the mark price for the market "ETH/DEC20" is "100291"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # 100291 is the new reference price, we get the following valid price ranges for the 2 triggers: [100135, 100447] & [100001, 100582]
    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100447 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100447 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100447"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # Now we should be after update and the bounds should change
    # T + 5s
    Then time is updated to "2020-10-16T00:00:26Z"

    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100448 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100448 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100448"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # Now, we have the following valid price ranges for the 2 triggers: [100213, 100525] & [100079, 100660]
    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 2      | 100213 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100050 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price for the market "ETH/DEC20" is "100448"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"


    # T + 2s
    Then time is updated to "2020-10-16T00:00:28Z"

    # Both triggers breached with market order -> 14s auction
    When traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type        | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 3      | 0     | 0                | TYPE_MARKET | TIF_FOK | ref-1     |


    And the mark price for the market "ETH/DEC20" is "100448"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 3s
    Then time is updated to "2020-10-16T00:00:33Z"

    And the mark price for the market "ETH/DEC20" is "100448"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 8s
    Then time is updated to "2020-10-16T00:00:43Z"

    And the mark price for the market "ETH/DEC20" is "100448"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"

    # 100213 is the new reference price, we get the following valid price ranges for the 2 triggers: [100057, 100369] & [99923, 100503]

    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100292 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100292 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    And the mark price for the market "ETH/DEC20" is "100292"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"


    # T + 12s
    Then time is updated to "2020-10-16T00:00:55Z"

     # Both triggers breached with persistent order -> auction with duration of 10s starts
    When traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100650 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100650 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |


    And the mark price for the market "ETH/DEC20" is "100292"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 6s
    Then time is updated to "2020-10-16T00:01:06Z"

    And the mark price for the market "ETH/DEC20" is "100292"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 1s
    Then time is updated to "2020-10-16T00:01:02Z"

    And the mark price for the market "ETH/DEC20" is "100292"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_MONITORING_AUCTION"

    # T + 8s
    Then time is updated to "2020-10-16T00:01:10Z"

    And the mark price for the market "ETH/DEC20" is "100650"

    And the trading mode for the market "ETH/DEC20" is "TRADING_MODE_CONTINUOUS"
