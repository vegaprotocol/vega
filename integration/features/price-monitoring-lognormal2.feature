Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [95878,104251], [90497,110401])

  Background:
    Given the markets start on "2020-10-16T00:00:00Z" and expire on "2020-12-31T23:59:59Z"
    And the price monitoring updated every "6000" seconds named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.95        | 240               |
      | 7200    | 0.999       | 360               |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | oracle config          |
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 3600             | default-none | my-price-monitoring | default-eth-for-future |
    And the following network parameters are set:
      | market.auction.minimumDuration |
      | 3600                           |
    And the oracles broadcast data signed with "0xDEADBEEF":
      | name             | value |
      | prices.ETH.value | 42    |

  Scenario: Auction triggered by 1st trigger (lower bound breached)
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|
      | aux2    | ETH   | 100000000000|

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "95878" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95877 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T03:04:03Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

  Scenario: Auction triggered by 1st trigger, upper bound
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "95878" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T03:04:03Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

  Scenario: Auction triggered by 1 trigger (upper bound breached)
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "95878" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T03:04:03Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

  Scenario: Auction triggered by both triggers (lower bound breached)
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "95878" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 90496 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 90496 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T02:04:03Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

    #T0 + 10min + 2 second opening auction
    Then time is updated to "2020-10-16T02:10:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 10min03s
    Then time is updated to "2020-10-16T03:10:03Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

  Scenario: Auction triggered by both triggers, upper bound
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "95878" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110402 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110402 | 0                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min + 2 second opening auction
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min03s
    Then time is updated to "2020-10-16T02:04:03Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

    #T0 + 10min + 2 second opening auction
    Then time is updated to "2020-10-16T02:10:01Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 10min03s
    Then time is updated to "2020-10-16T03:10:03Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

  Scenario: Auction triggered by 1st trigger (lower bound breached), extended by second (upper bound)
    Given the traders deposit on asset's general account the following amount:
      | trader  | asset | amount      |
      | trader1 | ETH   | 10000000000 |
      | trader2 | ETH   | 10000000000 |
      | trader3 | ETH   | 10000000000 |
      | trader4 | ETH   | 10000000000 |
      | aux     | ETH   | 100000000000|

     # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type        | tif     | 
      | aux     | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT  | TIF_GTC | 
      | aux     | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT  | TIF_GTC | 

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | trader3-1 |
      | trader4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | trader4-1 |
      | trader3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader3-2 |
      | trader4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | trader4-2 |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the traders cancel the following orders:
      | trader  | reference |
      | trader3 | trader3-1 |
      | trader4 | trader4-1 |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    Then time is updated to "2020-10-16T02:00:02Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "95878" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_FOK | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the traders place the following orders:
      | trader  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | cancel-me |
      | trader2 | ETH/DEC20 | buy  | 1      | 95877 | 0                | TYPE_LIMIT | TIF_FOK |           |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    Then the traders cancel the following orders:
      | trader  | reference |
      | trader1 | cancel-me |

    When the traders place the following orders:
      | trader  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC20 | sell | 1      | 110430 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | trader2 | ETH/DEC20 | buy  | 1      | 110430 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    #T0 + 4min01s
    Then time is updated to "2020-10-16T02:04:03Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

    #T0 + 10min
    Then time is updated to "2020-10-16T02:10:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "104251" for the market "ETH/DEC20"

    #T0 + 10min01sec
    Then time is updated to "2020-10-16T03:10:03Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110430" for the market "ETH/DEC20"
