Feature: Price monitoring triggers test on or around monitoring bounds with decimals

  Background:
    Given time is updated to "2020-10-16T00:00:00Z"
    And the following assets are registered:
      | id  | decimal places |
      | ETH | 10             |
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 30                |
    And the price monitoring named "my-updated-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.8         | 30                |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the markets:
      | id        | quote name | asset | risk model                    | margin calculator         | auction duration | fees         | price monitoring    | data source config     | decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | ETH        | ETH   | default-log-normal-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 5              | 0.001                  | 0                         |
    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    And the average block duration is "1"

  @PriceBounds
  Scenario: Trades below minimum price bound by 1 decimal trigger auction
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount                     |
      | party1 | ETH   | 1000000000000000000000     |
      | party2 | ETH   | 1000000000000000000000     |
      | aux    | ETH   | 1000000000000000000000     |
      | aux2   | ETH   | 1000000000000000000000     |
      | lpprov | ETH   | 90000000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount  | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price        | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 100000       | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | horizon | min bound  | max bound   | target stake    | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_CONTINUOUS | 5       | 9984405895 | 10015612113 | 743400000000000 | 9000000000000000 | 1             |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "10000000000" for the market "ETH/DEC20"
 
    #T0 + 10 min
    When the network moves ahead "1" blocks

    # Put in trade at min price bound -1 -> should trigger auction
    And the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | sell | 1      | 9984405894 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | buy  | 1      | 9984405894 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode                    | auction trigger       | target stake     | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1484481468319920 | 9000000000000000 | 1             |

    When the network moves ahead "1" blocks
    Then the mark price should be "10000000000" for the market "ETH/DEC20"

    # end of auction
    When the network moves ahead "30" blocks
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | target stake     | supplied stake   | open interest |
      | 9984405894 | TRADING_MODE_CONTINUOUS | 1484481468319920 | 9000000000000000 | 2             |

  @PriceBounds
  Scenario: Trades above maximum price bound by 1 decimal trigger auction
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount                     |
      | party1 | ETH   | 1000000000000000000000     |
      | party2 | ETH   | 1000000000000000000000     |
      | aux    | ETH   | 1000000000000000000000     |
      | aux2   | ETH   | 1000000000000000000000     |
      | lpprov | ETH   | 90000000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount  | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price        | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 100000       | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | horizon | min bound  | max bound   | target stake    | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_CONTINUOUS | 5       | 9984405895 | 10015612113 | 743400000000000 | 9000000000000000 | 1             |
 
    #T0 + 10 min
    When the network moves ahead "1" blocks

    # Put in trade at max price bound +1 -> should trigger auction
    And the parties place the following orders:
      | party  | market id | side | volume | price       | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | sell | 1      | 10015612114 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | buy  | 1      | 10015612114 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode                    | auction trigger       | target stake     | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1489121209109520 | 9000000000000000 | 1             |

    When the network moves ahead "1" blocks
    Then the mark price should be "10000000000" for the market "ETH/DEC20"

    # end of auction
    When the network moves ahead "30" blocks
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | target stake     | supplied stake   | open interest |
      | 10015612114 | TRADING_MODE_CONTINUOUS | 1489121209109520 | 9000000000000000 | 2             |

  @PriceBounds
  Scenario: Trades below minimum price bound by 1 after an update to the price monitoring parameters
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount                     |
      | party1 | ETH   | 1000000000000000000000     |
      | party2 | ETH   | 1000000000000000000000     |
      | aux    | ETH   | 1000000000000000000000     |
      | aux2   | ETH   | 1000000000000000000000     |
      | lpprov | ETH   | 90000000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount  | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price        | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 100000       | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | horizon | min bound  | max bound   | target stake    | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_CONTINUOUS | 5       | 9984405895 | 10015612113 | 743400000000000 | 9000000000000000 | 1             |
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the mark price should be "10000000000" for the market "ETH/DEC20"
 
    # 9000000001   11000000000
    # Update price monitoring bounds
    When the markets are updated:
      | id        | price monitoring            | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | my-updated-price-monitoring | 0.001                  | 0                         |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | horizon | min bound  | max bound   | target stake    | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_CONTINUOUS | 5       | 9000000001 | 11000000000 | 743400000000000 | 9000000000000000 | 1             |

    When the network moves ahead "1" blocks
    # Put in trade at min price bound -1 -> should trigger auction
    And the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | sell | 1      | 9000000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | buy  | 1      | 9000000000 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode                    | auction trigger       | target stake     | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1338120000000000 | 9000000000000000 | 1             |

    When the network moves ahead "1" blocks
    Then the mark price should be "10000000000" for the market "ETH/DEC20"

    # end of auction
    When the network moves ahead "30" blocks
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | target stake     | supplied stake   | open interest |
      | 9000000000 | TRADING_MODE_CONTINUOUS | 1338120000000000 | 9000000000000000 | 2             |

  @PriceBounds
  Scenario: Trades above maximum price bound by 1 decimal after market update
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount                     |
      | party1 | ETH   | 1000000000000000000000     |
      | party2 | ETH   | 1000000000000000000000     |
      | aux    | ETH   | 1000000000000000000000     |
      | aux2   | ETH   | 1000000000000000000000     |
      | lpprov | ETH   | 90000000000000000000000000 |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount  | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | buy  | BID              | 50         | 100    | submission |
      | lp1 | lpprov | ETH/DEC20 | 9000000000000000   | 0.1 | sell | ASK              | 50         | 100    | submission |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price        | resulting trades | type       | tif     |
      | aux   | ETH/DEC20 | buy  | 1      | 100000       | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 200000000000 | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC20 | buy  | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC20 | sell | 1      | 10000000000  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | horizon | min bound  | max bound   | target stake    | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_CONTINUOUS | 5       | 9984405895 | 10015612113 | 743400000000000 | 9000000000000000 | 1             |

    # Update price monitoring bounds
    When the markets are updated:
      | id        | price monitoring            | linear slippage factor | quadratic slippage factor |
      | ETH/DEC20 | my-updated-price-monitoring | 0.001                  | 0                         |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | horizon | min bound  | max bound   | target stake    | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_CONTINUOUS | 5       | 9000000001 | 11000000000 | 743400000000000 | 9000000000000000 | 1             |
 
    #T0 + 10 min
    When the network moves ahead "1" blocks

    # Put in trade at max price bound +1 -> should trigger auction
    And the parties place the following orders:
      | party  | market id | side | volume | price       | resulting trades | type       | tif     |
      | party1 | ETH/DEC20 | sell | 1      | 11000000001 | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/DEC20 | buy  | 1      | 11000000001 | 0                | TYPE_LIMIT | TIF_GTC |
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode                    | auction trigger       | target stake     | supplied stake   | open interest |
      | 10000000000 | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1635480000148680 | 9000000000000000 | 1             |

    When the network moves ahead "1" blocks
    Then the mark price should be "10000000000" for the market "ETH/DEC20"

    # end of auction
    When the network moves ahead "30" blocks
    Then the market data for the market "ETH/DEC20" should be:
      | mark price  | trading mode            | target stake     | supplied stake   | open interest |
      | 11000000001 | TRADING_MODE_CONTINUOUS | 1635480000148680 | 9000000000000000 | 2             |
