Feature: slippage_per_unit bug
  Background:
    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short = 3.5569036
    #risk factor long = 0.800728208
    And the price monitoring named "price-monitoring-1":
      | horizon  | probability | auction extension |
      | 72000000 | 0.99        | 3                 |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.5           | 2              | 3              |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 3              |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator   | auction duration | fees         | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/DEC23 | ETH        | USD   | log-normal-risk-model-1 | margin-calculator-1 | 1                | default-none | price-monitoring-1 | default-eth-for-future | 3              | 0                       | 1e2                    | 1e0                       |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | market.liquidity.stakeToCcyVolume       | 1     |

  Scenario: 001 Replicate a scenario from Lewis, linear slippage factor = 1e2, quadratic slippage factor = 1e0, 0019-MCAL-003
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount         |
      | traderA | USD   | 10000000000000 |
      | traderB | USD   | 3100000        |
      | traderC | USD   | 10000000000000 |
      | traderD | USD   | 10000          |
      | traderE | USD   | 10000          |
    When the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | traderB | ETH/DEC23 | 150000            | 0.001 | sell | ASK              | 100        | 20     | submission |
      | lp1 | traderB | ETH/DEC23 | 150000            | 0.001 | buy  | BID              | 100        | 20     | amendmend  |

    Then the parties place the following orders:
      | party   | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | traderA | ETH/DEC23 | buy  | 1      | 49    | 0                | TYPE_LIMIT | TIF_GTC | aux-b-5    |
      | traderB | ETH/DEC23 | sell | 1      | 350   | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1000 |
      | traderA | ETH/DEC23 | buy  | 1      | 350   | 0                | TYPE_LIMIT | TIF_GTC | aux-b-1    |
      | traderB | ETH/DEC23 | sell | 1      | 2000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
      | traderB | ETH/DEC23 | sell | 1      | 3000  | 0                | TYPE_LIMIT | TIF_GTC | aux-s-1    |
    When the opening auction period ends for market "ETH/DEC23"

    Then the order book should have the following volumes for market "ETH/DEC23":
      | side | price | volume |
      | buy  | 29    | 5173   |
      | buy  | 49    | 1      |
      | sell | 2000  | 1      |
      | sell | 2020  | 74     |
      | sell | 3000  | 1      |

    # traderB has both LP pegged orders, limit order, and positions
    # margin for pegged orders long and short: max(76*3.5569036,5173*0.800728208)*350=1449758.457
    # margin for short position: min(1*(2000-350)*1/1, 350*(1*1e2+1^2*1e0))+1*350*3.55690359157934000 =2894.916257
    # margin for the long position/orders is larger than the short size, so we take the margin for long side which is 1449759

    And the parties should have the following account balances:
      | party   | asset | market id | margin  | general | bond   |
      | traderB | USD   | ETH/DEC23 | 2899518 | 50482   | 150000 |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search  | initial | release |
      | traderB | ETH/DEC23 | 1449759     | 2174638 | 2899518 | 4349277 |

    And the following trades should be executed:
      | buyer   | price | size | seller  |
      | traderA | 350   | 1    | traderB |

    And the market data for the market "ETH/DEC23" should be:
      | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 12449        | 150000         | 1             |

    Then the order book should have the following volumes for market "ETH/DEC23":
      | side | price | volume |
      | buy  | 29    | 5173   |
      | buy  | 49    | 1      |
      | sell | 2000  | 1      |
      | sell | 2020  | 74     |
      | sell | 3000  | 1      |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | traderA | ETH/DEC23 | buy  | 111    | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | traderB | ETH/DEC23 | sell | 111    | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    # traderB has both LP pegged orders, limit order, and positions
    # margin for pegged orders long: 5173*0.801225765*50=207237.0441
    # margin for pegged+limit orders short:76*3.5569036*50=13516.23368
    # since the orders on the book is insufficient for order the short postition of traderB, the slippage_per_unit should set to infinit
    # margin for short positions: min(112*infinit, 50*(112*1e2+112^2*1e0))+112*50*3.55690359157934000 =1207118.66
    # margin_long = 207237.0441
    # margin_short= 13516.23368+1207118.66=1220634.894

    And the parties should have the following account balances:
      | party   | asset | market id | margin  | general | bond   |
      | traderB | USD   | ETH/DEC23 | 2441270 | 509024  | 150000 |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search  | initial | release |
      | traderB | ETH/DEC23 | 1220635     | 1830952 | 2441270 | 3661905 |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | traderA | 112    | -300           | 0            |
      | traderB | -112   | 300            | 0            |

    And the market data for the market "ETH/DEC23" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 50         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 199186       | 150000         | 112           |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general       | bond   |
      | traderA | USD   | ETH/DEC23 | 13754  | 9999999985946 | 0      |
      | traderB | USD   | ETH/DEC23 | 511138 | 2439156       | 150000 |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price       | resulting trades | type       | tif     |
      | traderC | ETH/DEC23 | sell | 120    | 45000000000 | 0                | TYPE_LIMIT | TIF_GTC |

    Then the order book should have the following volumes for market "ETH/DEC23":
      | side | price       | volume |
      | buy  | 29          | 5173   |
      | buy  | 49          | 1      |
      | sell | 2000        | 1      |
      | sell | 2020        | 74     |
      | sell | 3000        | 1      |
      | sell | 45000000000 | 120    |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | traderA | 112    | -300           | 0            |
      | traderB | -112   | 300            | 0            |

    # traderB has both LP pegged orders, limit order, and positions
    # margin for pegged orders long: 5173*0.801225765*50=207237.0441
    # margin for pegged orders short:76*3.5569036*50=13516.23368
    # margin for short positions: min(112*((2000-50)*1/112+(2020-50)*74/112+(3000-50)*1/112+(45000000000-50)*36/112), 50*(112*1e2+112^2*1e0))+112*50*3.55690359157934000 =1207118.66
    # margin_long = 207237.0441
    # margin_short= 13516.23368+1207118.66=1220634.894

    And the parties should have the following account balances:
      | party   | asset | market id | margin  | general       | bond   |
      | traderA | USD   | ETH/DEC23 | 13754   | 9999999985946 | 0      |
      | traderB | USD   | ETH/DEC23 | 2441270 | 509024        | 150000 |
      | traderC | USD   | ETH/DEC23 | 42684   | 9999999957316 | 0      |

    Then the parties should have the following margin levels:
      | party   | market id | maintenance | search  | initial | release |
      | traderB | ETH/DEC23 | 1220635     | 1830952 | 2441270 | 3661905 |

    And the market data for the market "ETH/DEC23" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 50         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 199186       | 150000         | 112           |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | traderA | 112    | -300           | 0            |
      | traderB | -112   | 300            | 0            |

    And the insurance pool balance should be "0" for the market "ETH/DEC23"

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | traderD | ETH/DEC23 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | traderE | ETH/DEC23 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond |
      | traderD | USD   | ETH/DEC23 | 82     | 9918    | 0    |
      | traderE | USD   | ETH/DEC23 | 4256   | 5743    | 0    |

    When the parties place the following orders with ticks:
      | party   | market id | side | volume | price | resulting trades | type       | tif     |
      | traderE | ETH/DEC23 | buy  | 1      | 50    | 0                | TYPE_LIMIT | TIF_GTC |
      | traderD | ETH/DEC23 | sell | 1      | 50    | 1                | TYPE_LIMIT | TIF_GTC |

    #for traderD and E, zero position and zero orders results in all zero margin levels
    And the parties should have the following account balances:
      | party   | asset | market id | margin | general | bond |
      | traderD | USD   | ETH/DEC23 | 0      | 9999    | 0    |
      | traderE | USD   | ETH/DEC23 | 0      | 9999    | 0    |

    Then the parties should have the following profit and loss:
      | party   | volume | unrealised pnl | realised pnl |
      | traderD | 0      | 0              | 0            |
      | traderE | 0      | 0              | 0            |


