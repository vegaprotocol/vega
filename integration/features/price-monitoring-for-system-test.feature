Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [99845,100156], [99711,100290])

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring updated every "4" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | maturity date        | risk model               | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | 2020-12-31T23:59:59Z | my-log-normal-risk-model | default-margin-calculator | 6                | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | name                           | value  |
      | market.auction.minimumDuration | 6      |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

  Scenario: Scenario for the system test with opening auction
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount       |
      | trader1 | ETH   | 100000000000 |
      | trader2 | ETH   | 100000000000 |
      | trader3 | ETH   | 100000000000 |
      | trader4 | ETH   | 100000000000 |
      | aux     | ETH   | 100000000000 |
      
     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 2      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 110000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    And the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference      |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1          |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-2          |
      | trader3 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader3_buy_1  |
      | trader4 | ETH/DEC20 | sell | 1      | 105000 | 0                | TYPE_LIMIT | TIF_GTC | trader4_sell_1 |

    Then the mark price should be "0" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    # T + 5s
    When time is updated to "2020-10-16T00:00:06Z"
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    # T + 1s
    When time is updated to "2020-10-16T00:00:07Z"
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # We've left opening auction, cancel the orders we had to place on the book to allow for this to happen
    And the traders cancel the following orders:
      | trader  | reference      |
      | trader3 | trader3_buy_1  |
      | trader4 | trader4_sell_1 |
      
    # 1st trigger breached with non-persistent order -> auction with initial duration of 6s starts
    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | trader1 | ETH/DEC20 | sell | 1      | 99844 | 0                | TYPE_LIMIT | TIF_GTC | trader1_sell_1 |
      | trader2 | ETH/DEC20 | buy  | 1      | 99844 | 0                | TYPE_LIMIT | TIF_GTC | ref-3          |

    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the traders cancel the following orders:
      | trader  | reference      |
      | trader1 | trader1_sell_1 |
      | trader2 | ref-3          |

    # T + 4s
    When time is updated to "2020-10-16T00:00:10Z"

    # 2nd trigger breached with persistent order -> auction extended by 8s (total auction time no 14s).
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100291 | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |

    # T + 10s (last second of the auciton)
    Then time is updated to "2020-10-16T00:00:20Z"

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 1s (auction ended)
    Then time is updated to "2020-10-16T00:00:22Z"

    And the mark price should be "100291" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # 100291 is the new reference price, we get the following valid price ranges for the 2 triggers: [100135, 100447] & [100001, 100582]
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100447 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100447 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100447" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # Now we should be after update and the bounds should change
    # T + 5s
    Then time is updated to "2020-10-16T00:00:26Z"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100448 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100448 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # Now, we have the following valid price ranges for the 2 triggers: [100213, 100525] & [100079, 100660]
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader2 | ETH/DEC20 | buy  | 2      | 100213 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100050 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"


    # T + 2s
    When time is updated to "2020-10-16T00:00:28Z"

    # Both triggers breached with market order -> 14s auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 3      | 1     | 0                | TYPE_LIMIT | TIF_GTC | ref-6     |


    And the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 3s
    When time is updated to "2020-10-16T00:00:33Z"

    Then the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the traders cancel the following orders:
      | trader  | reference |
      | trader1 | ref-6     |
    # T + 8s
    When time is updated to "2020-10-16T00:00:43Z"

    Then the mark price should be "100448" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # 100213 is the new reference price, we get the following valid price ranges for the 2 triggers: [100057, 100369] & [99923, 100503]

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100292 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100292 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |


    Then the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"


    # T + 12s
    When time is updated to "2020-10-16T00:00:55Z"

     # Both triggers breached with persistent order -> auction with duration of 10s starts
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100650 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100650 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |


    And the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 6s
    # T + 1s (min duration is 5 seconds, this test is broken)
    When time is updated to "2020-10-16T00:00:56Z"

    Then the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 1s (2 seconds)
    When time is updated to "2020-10-16T00:00:57Z"

    Then the mark price should be "100292" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # T + 8s (6s)
    When time is updated to "2020-10-16T00:01:12Z"

    Then the mark price should be "100650" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
