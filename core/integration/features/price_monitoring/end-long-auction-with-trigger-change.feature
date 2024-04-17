Feature: Confirm return to continuous trading during significant mark price change can be sped up with trigger modification mid-auction
  Background:
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
      | limits.markets.maxPeggedOrders          | 2     |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the price monitoring named "my-price-monitoring-1":
      | horizon | probability | auction extension |
      | 60      | 0.95        | 5                 |
      | 60      | 0.95        | 5                 |
      | 60      | 0.95        | 5                 |
      | 60      | 0.95        | 5                 |
      | 60      | 0.95        | 5                 |
    And the log normal risk model named "my-log-normal-risk-model":
      | risk aversion | tau                    | mu | r     | sigma |
      | 0.000001      | 0.00011407711613050422 | 0  | 0.016 | 2.0   |
    And the composite price oracles from "0xCAFECAFE1":
      | name    | price property   | price type   | price decimals |
      | oracle1 | prices.ETH.value | TYPE_INTEGER | 0              |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model               | margin calculator         | auction duration | fees         | price monitoring      | data source config     | linear slippage factor | quadratic slippage factor | sla params      | price type | decay weight | decay power | cash amount | source weights | source staleness tolerance | oracle1 | market type |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | my-log-normal-risk-model | default-margin-calculator | 5                | default-none | my-price-monitoring-1 | default-eth-for-future | 0.25                   | 0                         | default-futures | weight     | 0.1          | 0.5         | 500000      | 0,0,1,0        | 0s,0s,0h0m120s,0s          | oracle1 | future      |

    When the parties deposit on asset's general account the following amount:
      | party    | asset | amount       |
      | party1   | USD   | 100000000000 |
      | party2   | USD   | 100000000000 |
    And the parties place the following orders with ticks:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1            | ETH/FEB23 | buy  | 1      | 1000   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2            | ETH/FEB23 | sell | 1      | 1000   | 0                | TYPE_LIMIT | TIF_GTC |
    And the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 1000  | -1s         |
    And the network moves ahead "6" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 60      | 1000      | 995       | 1005      |

  Scenario: Mark price update without any intervention, book stuck at old price

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      |

    When the network moves ahead "100" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      |
 
    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |
    And the network moves ahead "25" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      |

    # In this scenario market will only leave the auction once the latest oracle-based mark-price candidate becomes stale
    When the network moves ahead "96" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | auction trigger             | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 60      | 1000      | 995       | 1005      |

  Scenario: Mark price update, followed by uncrossing trade near the new price

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      |

    When the parties place the following orders with ticks:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1            | ETH/FEB23 | buy  | 1      | 2004   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2            | ETH/FEB23 | sell | 1      | 2004   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "5" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # auction ends once auction extensions for all triggers elapse
    When the network moves ahead "21" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | last traded price | horizon | ref price | min bound | max bound |
      | 2000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 2004              | 60      | 2002      | 1992      | 2012      |

  Scenario: Mark price update, followed by trigger update

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound | auction end | 
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      | 5           |

    And the price monitoring named "my-price-monitoring-2":
      | horizon | probability | auction extension |
      | 30      | 0.95        | 1                 |
    
    # update trigger here so that there's only one
    When the markets are updated:
      | id        | price monitoring       | price type | source weights | source staleness tolerance | 
      | ETH/FEB23 | my-price-monitoring-2  | weight     | 0,0,1,0        | 0s,0s,0h0m120s,0s          |
    # first we get bounds based on default factors, once the state variable engine finishes its calculation we get the proper values
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 30      | 1000      | 900       | 1100      |

    # the last second of the original extensions
    And the network moves ahead "3" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 30      | 1000      | 997       | 1003      |

    # the updated extension triggered, no triggers left
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # market leaves the auction
    # mark price doesn't update as there weren't any trades so the reference price for price monitoring engine is still the last value it had (1000) 
    # which results in bounds that the mark price candidate (2000) is outside of
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | last traded price | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 1000              | 30      | 1000      | 997       | 1003      |

    # update trigger here so that there's only one
    When the markets are updated:
      | id        | price monitoring       | price type | source weights | source staleness tolerance |
      | ETH/FEB23 | my-price-monitoring-1  | weight     | 0,0,1,0        | 0s,0s,0h0m120s,0s          |

    # first we get bounds based on default factors, once the state variable engine finishes its calculation we get the proper values
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | last traded price | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 1000              | 60      | 1000      | 900       | 1100      |
    
    When the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | last traded price | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 1000              | 60      | 1000      | 995       | 1005      |

  Scenario: Mark price update, followed by trigger update (+ trade in auction, after market update)

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound | auction end | 
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      | 5           |

    And the price monitoring named "my-price-monitoring-2":
      | horizon | probability | auction extension |
      | 30      | 0.95        | 1                 |
    
    # update trigger here so that there's only one
    When the markets are updated:
      | id        | price monitoring       | price type | source weights | source staleness tolerance |
      | ETH/FEB23 | my-price-monitoring-2  | weight     | 0,0,1,0        | 0s,0s,0h0m120s,0s          |

    # the last second of the original extensions
    And the network moves ahead "3" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 30      | 1000      | 997       | 1003      |

    When the parties place the following orders with ticks:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1            | ETH/FEB23 | buy  | 1      | 2004   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2            | ETH/FEB23 | sell | 1      | 2004   | 0                | TYPE_LIMIT | TIF_GTC |

    # the updated extension triggered, no triggers left
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # market leaves the auction
    # All values received from oracle but not yet accepted as valid mark price get lost during the market update
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | last traded price | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 2004              | 30      | 2004      | 1997      | 2011      |

  Scenario: Mark price update, followed by trigger update (+ oracle value gets resubmitted after the market update)

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound | auction end | 
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 60      | 1000      | 995       | 1005      | 5           |

    And the price monitoring named "my-price-monitoring-2":
      | horizon | probability | auction extension |
      | 30      | 0.95        | 1                 |
    
    # update trigger here so that there's only one
    When the markets are updated:
      | id        | price monitoring       | price type | source weights | source staleness tolerance |
      | ETH/FEB23 | my-price-monitoring-2  | weight     | 0,0,1,0        | 0s,0s,0h0m120s,0s          |

    # the last second of the original extensions
    And the network moves ahead "3" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 30      | 1000      | 997       | 1003      |

    When the oracles broadcast data with block time signed with "0xCAFECAFE1":
      | name             | value | time offset |
      | prices.ETH.value | 2000  | -1s         |

    # the updated extension triggered, no triggers left
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE |

    # market leaves the auction
    # now we get stuck in an series of auctions (with uncrossing trades in between) until the mark price becomes stale
    When the parties place the following orders with ticks:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1            | ETH/FEB23 | buy  | 1      | 1002   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2            | ETH/FEB23 | sell | 1      | 1002   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | last traded price |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1002              | 

    When the parties place the following orders with ticks:
      | party             | market id | side | volume | price  | resulting trades | type       | tif     |
      | party1            | ETH/FEB23 | buy  | 1      | 1003   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2            | ETH/FEB23 | sell | 1      | 1003   | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger       | last traded price |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_PRICE | 1003              | 

    And the network moves ahead "118" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    | auction trigger             | last traded price | horizon | ref price | min bound | max bound |
      | 1000       | TRADING_MODE_CONTINUOUS         | AUCTION_TRIGGER_UNSPECIFIED | 1003              | 30      | 1003      | 1000      | 1006      | 