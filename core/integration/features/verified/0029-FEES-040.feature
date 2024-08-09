Feature: Fees calculations

  Background:
    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.5       | 0.6                |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 60      | 0.99        | 2                 |
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.2  | 0.1   | 100         | -100          | 0.1                    |

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
      | market.fee.factors.buybackFee           | 0.001 |
      | market.fee.factors.treasuryFee          | 0.002 |

    And the markets:
      | id        | quote name | asset | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC21 | ETH/USD    | USD   | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |

    And the average block duration is "2"
  Scenario: 001: Testing fees get collected when amended order trades (0029-FEES-005)
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount |
      | aux1    | USD   | 100000 |
      | aux2    | USD   | 100000 |
      | aux3    | USD   | 100000 |
      | aux4    | USD   | 100000 |
      | trader1 | USD   | 9000   |
      | trader2 | USD   | 300    |
      | trader3 | USD   | 2000   |
      | trader4 | USD   | 2000   |
      | trader5 | USD   | 5000   |
      | trader6 | USD   | 5000   |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.002 | submission |
      | lp1 | aux1  | ETH/DEC21 | 10000             | 0.002 | submission |
    When the network moves ahead "2" blocks

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux1  | ETH/DEC21 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC21 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | aux3  | ETH/DEC21 | buy  | 1      | 820   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux4  | ETH/DEC21 | sell | 1      | 1180  | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC21"
    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1000       | TRADING_MODE_CONTINUOUS |

    And the following trades should be executed:
      | buyer | price | size | seller |
      | aux1  | 1000  | 1    | aux2   |
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | aux1  | USD   | ETH/DEC21 | 540    | 89460   |

    #0029-FEES-040:In continuous trading mode, if the price taker has insufficient asset to cover the total fee in their general + margin account, then the trade should be discarded, the orders on the book that would have been hit should remain in place with previous remaining size intact and the incoming order should be rejected (not enough fees error).

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                      |
      | trader1 | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-01  |                                            |
      | trader2 | ETH/DEC21 | sell | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t2-s4-01  | party has insufficient funds to cover fees |

    And the orders should have the following status:
      | party   | reference | status          |
      | trader2 | t2-s4-01  | STATUS_REJECTED |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 1002  | 2      |
      | sell | 1002  | 0      |

    Then the parties cancel the following orders:
      | party   | reference |
      | trader1 | t1-b2-01  |

    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader3 | ETH/DEC21 | buy  | 2      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-03  |
      | trader4 | ETH/DEC21 | sell | 2      | 1101  | 0                | TYPE_LIMIT | TIF_GTC | t2-s4-04  |

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode                    | horizon | min bound | max bound |
      | 1000       | TRADING_MODE_MONITORING_AUCTION | 60      | 900       | 1100      |

    Then the parties cancel the following orders:
      | party   | reference |
      | trader3 | t1-b2-03  |
      | trader4 | t2-s4-04  |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 1002  | 0      |
      | buy  | 1101  | 0      |
      | sell | 1002  | 0      |
      | buy  | 1101  | 0      |

    #0029-FEES-041:In auction mode, if the price taker has insufficient asset to cover the total fee in their general + margin account, then the shortfall should be ignored, the orders should remain (instead of being rejected)
    When the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | trader1 | ETH/DEC21 | buy  | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t1-b2-05  |
      | trader2 | ETH/DEC21 | sell | 2      | 1002  | 0                | TYPE_LIMIT | TIF_GTC | t2-s4-06  |

    And the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 1002  | 2      |
      | buy  | 1101  | 0      |
      | sell | 1002  | 2      |
      | buy  | 1101  | 0      |

    When the network moves ahead "4" blocks

    And the market data for the market "ETH/DEC21" should be:
      | mark price | trading mode            |
      | 1002       | TRADING_MODE_CONTINUOUS |

    And the orders should have the following status:
      | party   | reference | status        |
      | trader2 | t2-s4-06  | STATUS_FILLED |

    #trader2 is closed out, after paying infra fee, trader2 does not have enough left to cover margin
    Then the parties should have the following account balances:
      | party   | asset | market id | margin | general |
      | trader1 | USD   | ETH/DEC21 | 1082   | 7308    |
      | trader2 | USD   | ETH/DEC21 | 0      | 0       |




