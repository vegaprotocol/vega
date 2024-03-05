Feature: Test order margin during continuous
  Background:
    # Set liquidity parameters to allow "zero" target-stake which is needed to construct the order-book defined in the ACs
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.2   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | position decimal places | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | 1                       | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 2     |
    Given the average block duration is "1"

  Scenario: Check order margin during continuous
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 100    | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | buy  | 20     | 15800  | 0                | TYPE_LIMIT | TIF_GFA | b-GFA-1   |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 20     | 15910  | 0                | TYPE_LIMIT | TIF_GTC | s-GTC-2   |
      | party1           | ETH/FEB23 | sell | 10     | 15920  | 0                | TYPE_LIMIT | TIF_GTC | s-GTC-1   |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 100    | 100100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "3" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 9540        | 10494  | 11448   | 13356   | cross margin | 0             | 0     |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.3           |       |

    #AC: 0019-MCAL-220, GFA order added during auction should not be used to count order margin in continuous
    #AC: 0019-MCAL-221, when party has no position, and place 2 short orders during auction, order margin should be updated
    #order margin short: (2*15910+1*15920)*0.3=14322
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |

    #AC: 0019-MCAL-222,When the party increases the order price during continunous, order margin should increase
    #order margin short: (2*15912+1*15920)*0.3=14323
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | s-GTC-2   | 15912 | 0          | TIF_GTC |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14323 |

    #AC: 0019-MCAL-223,When the party decreases the order price during continunous, order margin should decrease
    #order margin short: (2*15902+1*15920)*0.3=14317
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | s-GTC-2   | 15902 | 0          | TIF_GTC |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14317 |

    #AC: 0019-MCAL-224,When the party decreases the order volume during continunous, order margin should decrease
    #AC: 0019-MCAL-090,A feature test that checks margin in case market PDP > 0 is created and passes.
    #order margin short: (1*15902+1*15920)*0.3=9546
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | s-GTC-2   | 15902 | -10        | TIF_GTC |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 9546  |

    #AC: 0019-MCAL-225,When the party increases the order volume while decrease price during continunous, order margin should update accordingly
    #order margin short: (4*15900+1*15920)*0.3=23856
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | s-GTC-2   | 15900 | 30          | TIF_GTC |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 23856 |

    #AC: 0019-MCAL-226,When the party's order is partially filled during continunous, order margin should update accordingly
    #order margin short: (3*15900+1*15920)*0.3=19086
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 10     | 15900 | 1               | TYPE_LIMIT | TIF_GTC |           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 3180        | 0      | 3816    | 0       | isolated margin | 0.3           | 19086 |

    When the network moves ahead "1" blocks

    #AC: 0019-MCAL-227,When the party cancel one of the two orders during continunous, order margin should be reduced
    #order margin short: (3*15900+0*15920)*0.3=14310
    When the parties amend the following orders:
      | party  | reference | size delta | tif     | error |
      | party1 | s-GTC-1   | -10        | TIF_GTC |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 3180        | 0      | 3816    | 0       | isolated margin | 0.3           | 14310 |

    #AC: 0019-MCAL-228, place a GFA order duing continuous, order should be rejected
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference | error                                        |
      | party1 | ETH/FEB23 | buy  | 10     | 15800 | 0                | TYPE_LIMIT | TIF_GFA | GFA-1     | gfa order received during continuous trading |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 3180        | 0      | 3816    | 0       | isolated margin | 0.3           | 14310 |

    #AC: 0019-MCAL-229,When the party has position -1 and order -3, and new long order with size 1 will be offset
    #order margin short: 3*15900*0.3=14310
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 10     | 15800 | 0                | TYPE_LIMIT | TIF_GTC |           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 3180        | 0      | 3816    | 0       | isolated margin | 0.3           | 14310 |

    #AC: 0019-MCAL-230,When the party has position -1 and order -3, and new long orders with size 2 will be offset
    #order margin short: 3*15900*0.3=14310
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 10     | 15800 | 0                | TYPE_LIMIT | TIF_GTC |           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 3180        | 0      | 3816    | 0       | isolated margin | 0.3           | 14310 |

    #AC: 0019-MCAL-231,When the party has position -1 and order -3, and new long orders with size 3 will be offset
    #order margin short: 3*15900*0.3=14310
    #order margin short: 3*15800*0.3=14220
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 10     | 15800 | 0                | TYPE_LIMIT | TIF_GTC |           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 3180        | 0      | 3816    | 0       | isolated margin | 0.3           | 14310 |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |
