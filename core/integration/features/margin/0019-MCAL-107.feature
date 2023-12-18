Feature:  switch to isolated margin without position and with orders in auction
  Background:
    # switch between cross margin and isolated margin mode during auction
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 8     |
    Given the average block duration is "1"

  Scenario: 001 switch to isolated margin during auction
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 273500       |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 1      | 15800  | 0                | TYPE_LIMIT | TIF_GTC | p1-sell   |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price  | volume |
      | buy  | 14900  | 10     |
      | buy  | 15800  | 6      |
      | sell | 15800  | 1      |
      | sell | 15900  | 0      |
      | sell | 200000 | 1      |
    And the parties should have the following margin levels:
      | party           | market id | maintenance |
      | buySideProvider | ETH/FEB23 | 24380       |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 1896   | 270604  | 1000 |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 |
      | 0          | TRADING_MODE_OPENING_AUCTION |

    #AC0019-MCAL-107: switch to isolated margin without position and with orders with margin factor such that position margin is < initial should fail in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                                     |
      | party1 | ETH/FEB23 | isolated margin | 0.1           | Margin factor (0.1) must be greater than max(riskFactorLong (0.1), riskFactorShort (0.1)) |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 1580        | 1738   | 1896    | 2212    | cross margin | 0             | 0     |

    #AC0019-MCAL-123:switch to isolated margin without position and with orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |       |

    # @jiajia
    # the party has no position so position initial/maintenance is 0...
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.6           | 9480  |

    #AC0019-MCAL-131:increase margin factor in isolated margin without position and with orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.7           |       |

    # same, the don't have position so their maintenance margin (which doesn't consider orders) is 0
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.7           | 11060 |

    And the orders should have the following status:
      | party  | reference | status        |
      | party1 | p1-sell   | STATUS_ACTIVE |

    When the parties cancel the following orders:
      | party  | reference |
      | party1 | p1-sell   |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | p1-sell   | STATUS_CANCELLED |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 |
      | 0          | TRADING_MODE_OPENING_AUCTION |

    #AC0019-MCAL-135:switch to cross margin without position and no orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error                      |
      | party1 | ETH/FEB23 | cross margin |               | no market observable price |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.7           | 0     |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC |           |

    #AC0019-MCAL-103:switch back to cross margin with no position and no order in continuous mode in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor |
      | party1 | ETH/FEB23 | cross margin |               |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | cross margin | 0.7           | 0     |


