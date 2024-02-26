Feature: Test order margin during auction
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
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 2     |
    Given the average block duration is "1"

  Scenario: Check order margin during openning auction
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 2      | 15910  | 0                | TYPE_LIMIT | TIF_GTC | s-GTC-2   |
      | party1           | ETH/FEB23 | sell | 1      | 15920  | 0                | TYPE_LIMIT | TIF_GTC | s-GTC-1   |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 |
      | 0          | TRADING_MODE_OPENING_AUCTION |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 4774        | 5251   | 5728    | 6683    | cross margin | 0             | 0     |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.3           |       |

    #AC: 0019-MCAL-200, when party has no position, and place 2 short orders during auction, order margin should be updated
    #order margin short: (2*15910+1*15920)*0.3=14322
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |

    #AC: 0019-MCAL-201, when party has no position, and place short orders size -3 during auction, and long order size 1 which can offset, order margin should be updated using max(price, markPrice, indicativePrice)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | b-GTC-1   |

    #order margin short: (2*15910+1*15920)*0.3=14322
    #order margin long: 1*15800*0.3=5750
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |

    #AC: 0019-MCAL-202, when party has no position, and place short orders size -3 during auction, and long orders size 2 which can offset, order margin should be updated using max(price, markPrice, indicativePrice)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | b-GTC-2   |

    #order margin short: (2*15910+1*15920)*0.3=14322
    #order margin long: 2*15800*0.3=9480
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |

    #AC: 0019-MCAL-203, when party has no position, and place short orders size -3 during auction, and long orders size 3 which can offset, order margin should be updated using max(price, markPrice, indicativePrice)
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | b-GTC-3   |

    #order margin short: (2*15910+1*15920)*0.3=14322
    #order margin long: 3*15800*0.3=14220
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |

    #AC: 0019-MCAL-204, when party has no position, and place short orders size -3 during auction, and long orders size 4, which is over the offset size, order margin should be updated using max(price, markPrice, indicativePrice)
    #order margin short: (2*15910+1*15920)*0.3=14322
    #order margin long: 4*15900*0.3=19080
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | b-GTC-4   |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 19080 |

    #AC: 0019-MCAL-205,When the party changes the order price during auction, order margin should be updated using max(price, markPrice, indicativePrice)
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | b-GTC-3   | 15750 | 0          | TIF_GTC |       |
    When the network moves ahead "1" blocks
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 19080 |

    #AC: 0019-MCAL-206,When the party reduces the order size only during auction, the order margin should be reduced 
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | b-GTC-4   | 15800 | -1         | TIF_GTC |       |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | b-GTC-4   | STATUS_CANCELLED |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |
    When the network moves ahead "2" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |

    #order margin long: 3*15800*0.3=14220
    #order margin short: (2*15910+1*15920)*0.3=14322
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 14322 |

    #AC: 0019-MCAL-207, when party has no position, and place 2 short orders size 3 and 4 long orders of size 4, which is over the offset size, order margin should be updated using max(price, markPrice, indicativePrice)
    #order margin long: (3*15800+15750)*0.3=18945
    #order margin short: (2*15910+1*15920)*0.3=14322
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | b-GTC-4   |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.3           | 18945 |

