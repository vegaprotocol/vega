Feature:  switch to isolated margin with position and without orders during continuous
  Background:
    # switch between cross margin and isolated margin mode during auction
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the price monitoring named "my-price-monitoring":
      | horizon | probability | auction extension |
      | 5       | 0.95        | 6                 |
      | 10      | 0.99        | 8                 |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring    | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | my-price-monitoring | default-eth-for-future | 0.25                   | 0                         | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 1     |
    Given the average block duration is "1"

  Scenario: 001 switch to isolated margin during auction
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 20908        |
      | party2           | USD   | 84110        |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15600  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 3      | 15800  | 0                | TYPE_LIMIT | TIF_GTC | p1-sell   |
      | sellSideProvider | ETH/FEB23 | sell | 3      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 1000           | 3             |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 19908  | 0       | 1000 |

    #maintenance margin: min((200000-15800),15800*0.25)+0.1*15800=5530
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 16590       | 18249  | 19908   | 23226   | cross margin | 0             | 0     |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                                                                   |
      | party1 | ETH/FEB23 | isolated margin | 0.1           | margin factor (0.1) must be greater than max(riskFactorLong (0.1), riskFactorShort (0.1)) + linearSlippageFactor (0.25) |

    #AC0019-MCAL-108: switch to isolated margin with position and with orders with margin factor such that position margin is < initial should fail in continuous
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                        |
      | party1 | ETH/FEB23 | isolated margin | 0.36          | required position margin must be greater than initial margin |

    #AC0019-MCAL-112: switch to isolated margin with position and no orders with margin factor such that there is insufficient balance in the general account in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                      |
      | party1 | ETH/FEB23 | isolated margin | 0.8           | insufficient balance in general account to cover for required order margin |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1           | ETH/FEB23 | buy  | 2      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | p1-sell-1 |
      | sellSideProvider | ETH/FEB23 | sell | 2      | 15800 | 1                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 6636   | 13272   | 1000 |

    #AC0019-MCAL-120: switch to isolated margin with position and no orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.5           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.5           | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 7900   | 12008   | 1000 |

    #AC0019-MCAL-128: increase margin factor in isolated margin with position and no orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.7           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.7           | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 11060  | 8848    | 1000 |

    #AC0019-MCAL-128: decrease margin factor in isolated margin with position and no orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.6           | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 9480   | 10428   | 1000 |

    #AC0019-MCAL-136: switch to cross margin with position and no orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party1 | ETH/FEB23 | cross margin |               |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 6083   | 6636    | 7742    | cross margin | 0             | 0     |

    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 9480   | 10428   | 1000 |
