Feature: When a market's trigger and extension_trigger are set to represent that the market went into auction due to the price monitoring mechanism and was later extended by the same mechanism and the auction is meant to finish at 11am, but now a long block auction is being triggered so that it ends at 10am then this market is unaffected in any way. (0094-PRAC-008) When market is in a price monitoring auction which is meant to finish at 10am, but prior to that time a long block auction finishing at 11am gets triggered then the market stays in auction till 11am, it's auction trigger is listed as price monitoring auction and it's extension trigger is listed as long block auction. (0094-PRAC-006).
         

  Background:
    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 5              |
    And the long block duration table is:
      | threshold | duration |
      | 3s        | 1m       |
      | 40s       | 10m      |
      | 2m        | 1h       |
    And the price monitoring named "my-price-monitoring-2":
      | horizon | probability | auction extension |
      | 360     | 0.95        | 3600              |
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 360     | 0.95        | 61                |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the simple risk model named "my-simple-risk-model":
      | long                   | short                  | max move up | min move down | probability of trading |
      | 0.08628781058136630000 | 0.09370922348428490000 | -1          | -1            | 0.2                    |
    And the fees configuration named "my-fees-config":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    # create 2 markets
    And the markets:
      | id        | quote name | asset | risk model           | margin calculator         | auction duration | fees           | price monitoring      | data source config     | decimal places | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | my-price-monitoring-2 | default-eth-for-future | 2              | 0.25                   | 0                         | default-futures |
      | ETH/DEC20 | ETH        | ETH   | my-simple-risk-model | default-margin-calculator | 1                | my-fees-config | my-price-monitoring   | default-eth-for-future | 2              | 0.25                   | 0                         | default-futures |
    And the following network parameters are set:
      | name                           | value |
      | limits.markets.maxPeggedOrders | 2     |
    And the average block duration is "1"
    And the parties deposit on asset's general account the following amount:
      | party   | asset | amount        |
      | party1  | ETH   | 1000000000000 |
      | party2  | ETH   | 1000000000000 |
      | party3  | ETH   | 1000000000000 |
      | party4  | ETH   | 1000000000000 |
      | party5  | ETH   | 1000000000000 |
      | party6  | ETH   | 1000000000000 |
      | lpprov1 | ETH   | 1000000000000 |
      | lpprov2 | ETH   | 1000000000000 |
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov1 | ETH/DEC20 | 1873996252        | 0.1 | submission |
      | lp2 | lpprov2 | ETH/DEC19 | 1873996252        | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party   | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lpprov1 | ETH/DEC20 | 2         | 1                    | buy  | BID              | 50     | 100    |
      | lpprov1 | ETH/DEC20 | 2         | 1                    | sell | ASK              | 50     | 100    |
      | lpprov2 | ETH/DEC19 | 2         | 1                    | buy  | MID              | 50     | 100    |
      | lpprov2 | ETH/DEC19 | 2         | 1                    | sell | MID              | 50     | 100    |

  @LBA
  Scenario: When market is in a price monitoring auction which is meant to finish at 10am, but prior to that time a long block auction finishing at 11am gets triggered then the market stays in auction till 11am, it's auction trigger is listed as price monitoring auction and it's extension trigger is listed as long block auction. (0094-PRAC-006). 0094-PRAC-008: Long block auction exceeds the price monitoring auction duration, the auction gets extended.
    # place orders and generate trades - slippage 100
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 999500  | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party1 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC20 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | party3 | ETH/DEC19 | buy  | 1      | 999500  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party3 | ETH/DEC19 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-2    |
      | party4 | ETH/DEC19 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 1873996252     | 937000000    |
    And the market data for the market "ETH/DEC19" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 1873996252     | 937000000    |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    #And debug detailed orderbook volumes for market "ETH/DEC20"
    And the order book should have the following volumes for market "ETH/DEC20":
      | volume | price   | side |
      | 2      | 999400  | buy  |
      | 1      | 999500  | buy  |
      | 1      | 1000000 | sell |
      | 2      | 1000100 | sell |
    #And debug detailed orderbook volumes for market "ETH/DEC19"
    And the order book should have the following volumes for market "ETH/DEC19":
      | volume | price   | side |
      | 1      | 999500  | buy  |
      | 2      | 999650  | buy  |
      | 2      | 999850  | sell |
      | 1      | 1000000 | sell |

    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party1 | 1000000 | 1    | party2 |
      | party3 | 1000000 | 1    | party4 |
    And the mark price should be "1000000" for the market "ETH/DEC19"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000000    | TRADING_MODE_CONTINUOUS | 360     | 1000000   | 999999    | 937000000    | 1873996252     | 1             |

    When the network moves ahead "10" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC20 | buy  | 1      | 999998 | 0                | TYPE_LIMIT | TIF_GTC | t5-b-1    |
      | party6 | ETH/DEC20 | sell | 1      | 999998 | 0                | TYPE_LIMIT | TIF_GTC | t6-s-1    |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 360     | 1000000   | 999999    | 1873996252   | 1873996252     | 1             | 61          |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest | auction end |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 1873996252   | 1873996252     | 1             | 61          |

    When the previous block duration was "90s"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC19"

    # We know what the volume on the books look like, but let's submit some orders that will trade regardless
    # And we'll see no trades happen
    When the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 999998 | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | party2 | ETH/DEC20 | sell | 1      | 999998 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |

    # ETH/DEC19 - demonstrates 0094-PRAC-008
    # ETH/DEC20 - demonstrates 0094-PRAC-006
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC19"
    And the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest | auction end | extension trigger          |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 2810994378   | 1873996252     | 1             | 602         | AUCTION_TRIGGER_LONG_BLOCK |

    # move ahead another minute
    When the network moves ahead "1" blocks
    # This is strange, it looks as though the trade went through at the end of the auction, but in doing so triggered a second auction?
    Then the market data for the market "ETH/DEC20" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest | extension trigger          |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 2810994378   | 1873996252     | 1             | AUCTION_TRIGGER_LONG_BLOCK |

    When the network moves ahead "9m50s" with block duration of "2s"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC19"

    # still in auction, but if we move ahead...
    When the network moves ahead "11" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC20"
    # now move further ahead to leave price auction
    # We have moved 1 blocks + 9m50s (9m51) + 11 blocks for a total of 10m2s, the total auction duration
    # was 602s, or 10m2s. Leaving the auction will trigger an extension of another 61 seconds
    # So the total time in auction would be 11m2s. At this point we're still 61s short.
    When the network moves ahead "61" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the following trades should be executed:
      | buyer  | price  | size | seller |
      | party5 | 999998 | 1    | party6 |
      | party1 | 999998 | 1    | party2 |


  @LBA
  Scenario: 0094-PRAC-008: Long block auction does not exceed the price monitoring auction duration, the auction does not get extended.
    # place orders and generate trades - slippage 100
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC20 | buy  | 1      | 999500  | 0                | TYPE_LIMIT | TIF_GTC | t1-b-1    |
      | party1 | ETH/DEC20 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GFA | t1-b-2    |
      | party2 | ETH/DEC20 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t2-s-1    |
      | party3 | ETH/DEC19 | buy  | 1      | 999500  | 0                | TYPE_LIMIT | TIF_GTC | t3-b-1    |
      | party3 | ETH/DEC19 | buy  | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t3-b-2    |
      | party4 | ETH/DEC19 | sell | 2      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | t4-s-1    |
    Then the market data for the market "ETH/DEC20" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 1873996252     | 937000000    |
    And the market data for the market "ETH/DEC19" should be:
      | trading mode                 | supplied stake | target stake |
      | TRADING_MODE_OPENING_AUCTION | 1873996252     | 937000000    |

    When the network moves ahead "2" blocks
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    #And debug detailed orderbook volumes for market "ETH/DEC20"
    And the order book should have the following volumes for market "ETH/DEC20":
      | volume | price   | side |
      | 2      | 999400  | buy  |
      | 1      | 999500  | buy  |
      | 1      | 1000000 | sell |
      | 2      | 1000100 | sell |
    #And debug detailed orderbook volumes for market "ETH/DEC19"
    And the order book should have the following volumes for market "ETH/DEC19":
      | volume | price   | side |
      | 1      | 999500  | buy  |
      | 2      | 999650  | buy  |
      | 2      | 999850  | sell |
      | 1      | 1000000 | sell |

    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party1 | 1000000 | 1    | party2 |
      | party3 | 1000000 | 1    | party4 |
    And the mark price should be "1000000" for the market "ETH/DEC20"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000000    | TRADING_MODE_CONTINUOUS | 360     | 1000000   | 999999    | 937000000    | 1873996252     | 1             |

    When the network moves ahead "10" blocks
    Then the parties place the following orders:
      | party  | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | party5 | ETH/DEC19 | buy  | 1      | 999998 | 0                | TYPE_LIMIT | TIF_GTC | t5-b-1    |
      | party6 | ETH/DEC19 | sell | 1      | 999998 | 0                | TYPE_LIMIT | TIF_GTC | t6-s-1    |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                    | horizon | min bound | max bound | target stake | supplied stake | open interest | auction end |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 360     | 1000000   | 999999    | 1873996252   | 1873996252     | 1             | 3600        |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest | auction end |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 1873996252   | 1873996252     | 1             | 3600        |

    When the previous block duration was "40s"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC20"

    # We know what the volume on the books look like, but let's submit some orders that will trade regardless
    # And we'll see no trades happen
    When the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference |
      | party1 | ETH/DEC19 | buy  | 1      | 999998  | 0                | TYPE_LIMIT | TIF_GTC | t1-b-3    |
      | party2 | ETH/DEC19 | sell | 1      | 999998  | 0                | TYPE_LIMIT | TIF_GTC | t2-s-2    |
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC20"

    # the monitoring auction is not extended
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/DEC19" should be:
      | mark price | trading mode                    | target stake | supplied stake | open interest | auction end | extension trigger           |
      | 1000000    | TRADING_MODE_MONITORING_AUCTION | 2810994378   | 1873996252     | 1             | 3600        | AUCTION_TRIGGER_UNSPECIFIED |

    When the network moves ahead "9m50s" with block duration of "2s"
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_LONG_BLOCK_AUCTION" for the market "ETH/DEC20"

    # still in auction, 1m10 seconds later, though:
    When the network moves ahead "71" blocks
    Then the trading mode should be "TRADING_MODE_MONITORING_AUCTION" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"

    # we still have to leave the monitoring auction, so let's move ahead a bit
    When the network moves ahead "2h" with block duration of "10m"
    Then the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC20"
    And the following trades should be executed:
      | buyer  | price   | size | seller |
      | party5 | 999998  | 1    | party6 |
      | party1 | 999998  | 1    | party2 |

