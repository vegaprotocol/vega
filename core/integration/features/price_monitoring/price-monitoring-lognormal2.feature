Feature: Price monitoring test using forward risk model (bounds for the valid price moves around price of 100000 for the two horizons are: [95878,104251], [90497,110401])

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.95        | 240               |
      | 7200    | 0.999       | 360               |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | my-log-normal-risk-model | default-margin-calculator | 3600             | default-none | my-price-monitoring | default-eth-for-future | 1E-4                   | 1E-4                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 100   |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  Scenario: Auction triggered by 1st trigger (lower bound breached)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | aux2   | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100000     | 95878             | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100000     | 104251            | TRADING_MODE_CONTINUOUS |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 100000     | TRADING_MODE_CONTINUOUS | 3600    | 95878     | 104251    |
      | 100000     | TRADING_MODE_CONTINUOUS | 7200    | 90497     | 110401    |

    # T0
    Then time is updated to "2020-10-16T02:00:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "104251" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min + 1 second
    Then time is updated to "2020-10-16T02:04:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "95877" for the market "ETH/DEC20"

  Scenario: Auction triggered by 1st trigger, upper bound
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # T0
    Then time is updated to "2020-10-16T02:00:00Z"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100000     | 95878             | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 100000     | 104251            | TRADING_MODE_CONTINUOUS |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 100000     | TRADING_MODE_CONTINUOUS | 3600    | 95878     | 104251    |
      | 100000     | TRADING_MODE_CONTINUOUS | 7200    | 90497     | 110401    |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode                    |
      | 100000     | 104251            | TRADING_MODE_MONITORING_AUCTION |

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min04s
    Then time is updated to "2020-10-16T03:04:04Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 104252     | 104252            | TRADING_MODE_CONTINUOUS |

  Scenario: Non-opening auction can end with wash trades
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    Then the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    When the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # T0
    Then time is updated to "2020-10-16T02:00:00Z"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the mark price should be "100000" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | last traded price | trading mode            |
      | 95878      | 95878             | TRADING_MODE_CONTINUOUS |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |
    Then the mark price should be "104251" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 104251     | TRADING_MODE_CONTINUOUS | 3600    | 95878     | 104251    |
      | 104251     | TRADING_MODE_CONTINUOUS | 7200    | 90497     | 110401    |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-3     |
      | party2 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-4     |

    Then the parties cancel the following orders:
      | party  | reference |
      | party2 | ref-4     |

    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | open interest |
      | 104251     | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 4             |

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    # Assure no change in target stake due to wash trade
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | open interest |
      | 104251     | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 4             |


    # Submit trade so that auction uncrosses with wash trade
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 104252 | 0                | TYPE_LIMIT | TIF_GTC | ref-5     |

    # Assure no change in target stake due to wash trade
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | open interest |
      | 104251     | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 4             |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -3     | -12624         | 0            |
      | party2 | 3      | 12624          | 0            |
      | party3 | -1     | -4251          | 0            |
      | party4 | 1      | 4251           | 0            |
      | aux    | 0      | 0              | 0            |

    #T0 + 4min04s
    Then time is updated to "2020-10-16T03:04:04Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # Assure uncrossing trade was indeed a wash trade
    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party1 | 104252 | 1    | party1 |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | -3     | -8418          | -4209        |
      | party2 | 3      | 12627          | 0            |
      | party3 | -1     | -4252          | 0            |
      | party4 | 1      | 4252           | 0            |
      | aux    | 0      | 0              | 0            |

    And the mark price should be "104252" for the market "ETH/DEC20"

  Scenario: Auction triggered by 1 trigger (upper bound breached)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104253 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104253 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min01s
    Then time is updated to "2020-10-16T03:04:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "104253" for the market "ETH/DEC20"

  Scenario: Auction triggered by both triggers (lower bound breached)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0
    Then time is updated to "2020-10-16T02:00:00Z"

    Then the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 90496 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 90496 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min01s
    Then time is updated to "2020-10-16T02:04:01Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10min
    Then time is updated to "2020-10-16T02:10:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T02:10:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "90496" for the market "ETH/DEC20"

  Scenario: Auction triggered by both triggers, upper bound
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0
    Then time is updated to "2020-10-16T02:00:00Z"

    Then the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the market data for the market "ETH/DEC20" should be:
      | horizon | min bound | max bound |
      | 3600    | 95878     | 104251    |
      | 7200    | 90497     | 110401    |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 110437 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 110437 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the market data for the market "ETH/DEC20" should be:
      | horizon | min bound | max bound |
      | 7200    | 90526     | 110436    |

    #T0 + 4min01s
    Then time is updated to "2020-10-16T02:04:01Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10min
    Then time is updated to "2020-10-16T02:10:00Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 10min01s
    Then time is updated to "2020-10-16T03:10:01Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    And the mark price should be "110437" for the market "ETH/DEC20"

  Scenario: Auction triggered by 1st trigger (lower bound breached), extended by second (upper bound)
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount       |
      | party1 | ETH   | 10000000000  |
      | party2 | ETH   | 10000000000  |
      | party3 | ETH   | 10000000000  |
      | party4 | ETH   | 10000000000  |
      | aux    | ETH   | 100000000000 |
      | lpprov | ETH   | 100000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 90000000          | 0.1 | sell | ASK              | 50         | 100    | submission |

    Then the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | aux   | ETH/DEC20 | 614212            | 0.001 | buy  | BID              | 1          | 10     | submission |
      | lp1 | aux   | ETH/DEC20 | 614212            | 0.001 | sell | ASK              | 1          | 10     | amendment  |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    Then the parties place the following orders:
      | party | market id | side | volume | price  | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 1      | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |

    # Trigger an auction to set the mark price
    And the trading mode should be "TRADING_MODE_OPENING_AUCTION" for the market "ETH/DEC20"
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC20 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC | party3-1  |
      | party4 | ETH/DEC20 | buy  | 1      | 80000  | 0                | TYPE_LIMIT | TIF_GTC | party4-1  |
      | party3 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party3-2  |
      | party4 | ETH/DEC20 | buy  | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GFA | party4-2  |
    Then the opening auction period ends for market "ETH/DEC20"
    And the mark price should be "100000" for the market "ETH/DEC20"
    Then the parties cancel the following orders:
      | party  | reference |
      | party3 | party3-1  |
      | party4 | party4-1  |

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 100000 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    Then time is updated to "2020-10-16T02:00:02Z"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95878 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 95878 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 104251 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 104251 | 1                | TYPE_LIMIT | TIF_GTC | ref-2     |

    And the mark price should be "100000" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound |
      | 100000     | TRADING_MODE_CONTINUOUS | 3600    | 95878     | 104251    |
      | 100000     | TRADING_MODE_CONTINUOUS | 7200    | 90497     | 110401    |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC | cancel-me |
      | party2 | ETH/DEC20 | buy  | 1      | 95877 | 0                | TYPE_LIMIT | TIF_GTC |           |

    And the mark price should be "100000" for the market "ETH/DEC20"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    #T0 + 4min
    Then time is updated to "2020-10-16T02:04:02Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    Then the parties cancel the following orders:
      | party  | reference |
      | party1 | cancel-me |

    # Additional sell volume prevents liquidity extension
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | sell | 1      | 110431 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party1 | ETH/DEC20 | sell | 1      | 110430 | 0                | TYPE_LIMIT | TIF_GTC | ref-1     |
      | party2 | ETH/DEC20 | buy  | 1      | 110430 | 0                | TYPE_LIMIT | TIF_GTC | ref-2     |

    #T0 + 4min01s
    Then time is updated to "2020-10-16T02:04:03Z"

    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"

    And the mark price should be "100000" for the market "ETH/DEC20"

    #T0 + 10min
    Then time is updated to "2020-10-16T02:10:02Z"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | auction trigger       | extension trigger     | target stake | supplied stake | open interest |
      | 100000     | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | AUCTION_TRIGGER_PRICE | 614211       | 614212         | 4             |

    #T0 + 10min01sec
    Then time is updated to "2020-10-16T02:10:03Z"

    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | auction trigger             | extension trigger           | target stake | supplied stake | open interest |
      | 110430     | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | AUCTION_TRIGGER_UNSPECIFIED | 614211       | 614212         | 5             |
