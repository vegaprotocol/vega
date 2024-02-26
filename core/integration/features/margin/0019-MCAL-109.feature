Feature:  switch to isolated margin without position and with orders during auction
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
      | party1           | USD   | 28550        |
      | party2           | USD   | 84110        |
      | party3           | USD   | 84110        |
      | party4           | USD   | 84110        |
      | party5           | USD   | 84110        |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15600  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party4           | ETH/FEB23 | sell | 1      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 1      | 15800  | 0                | TYPE_LIMIT | TIF_GTC | p1-sell   |
      | party2           | ETH/FEB23 | sell | 1      | 15802  | 0                | TYPE_LIMIT | TIF_GTC | p1-sell   |
      | party1           | ETH/FEB23 | sell | 2      | 15804  | 0                | TYPE_LIMIT | TIF_GTC | p1-sell   |
      | sellSideProvider | ETH/FEB23 | sell | 3      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15800      | TRADING_MODE_CONTINUOUS | 5       | 15701     | 15899     | 0            | 1000           | 2             |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 5689   | 21861   | 1000 |

    #maintenance margin: min((15802-15800),15800*0.1)+0.1*15800+2*0.1*15800=4742
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 4742        | 5216   | 5690    | 6638    | cross margin | 0             | 0     |

    #AC0019-MCAL-134:switch to cross margin without position and no orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party5 | ETH/FEB23 | isolated margin | 0.4           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party5 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 0     |
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party5 | ETH/FEB23 | cross margin | 0.4           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party5 | ETH/FEB23 | 0           | 0      | 0       | 0       | cross margin | 0             | 0     |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party5 | ETH/FEB23 | buy  | 1      | 14800 | 0                | TYPE_LIMIT | TIF_GTC |           |

    #AC0019-MCAL-130: increase margin factor in isolated margin without position and with orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party5 | ETH/FEB23 | isolated margin | 0.4           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party5 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 5920  |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party5 | ETH/FEB23 | isolated margin | 0.5           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party5 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.5           | 7400  |

    #now trigger price monitoring auction
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15600 | 0                | TYPE_LIMIT | TIF_GTC |           |
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15800      | TRADING_MODE_MONITORING_AUCTION |

    #AC0019-MCAL-109: switch to isolated margin with position and with orders with margin factor such that position margin is < initial should fail in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                        |
      | party1 | ETH/FEB23 | isolated margin | 0.2           | required position margin must be greater than initial margin |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 5689   | 21861   | 1000 |

    #AC0019-MCAL-122: switch to isolated margin without position and with orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party2 | ETH/FEB23 | isolated margin | 0.4           |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party2 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 6320  |

    #AC0019-MCAL-138: switch to cross margin without position and with orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party2 | ETH/FEB23 | cross margin |               |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party2 | ETH/FEB23 | 1581        | 1739   | 1897    | 2213    | cross margin | 0             | 0     |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 4742        | 5216   | 5690    | 6638    | cross margin | 0             | 0     |

    When the network moves ahead "1" blocks
    #AC0019-MCAL-142:switch to isolated margin with position and with orders with margin factor such that there is insufficient balance in the general account in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                      |
      | party1 | ETH/FEB23 | isolated margin | 0.9           | insufficient balance in general account to cover for required order margin |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 5689   | 21861   | 1000 |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 4742        | 5216   | 5690    | 6638    | cross margin | 0             | 0     |

    #AC0019-MCAL-123:switch to isolated margin with position and with orders successful in auction
    #required position margin: 15800*0.6=9480
    #maintenance margin for position: 15800*0.25+0.1*15800=5530
    #initial margin for position: 5530*1.2=6636
    # @jiajia there's insufficient in the general account here:
    # for this switch to work they need 22755 in their general account, they only have 21861
    # requiredPositionMargin 9480
    # requireOrderMargin 18964.8
    # total required: 9480 + 18964 - 5689 = 22,755
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                      |
      | party1 | ETH/FEB23 | isolated margin | 0.6           | insufficient balance in general account to cover for required order margin |

    #AC0019-MCAL-124:switch to isolated margin with position and with orders successful in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.55          |       |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 8690   | 1476    | 1000 |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.55          | 17384 |

    #AC0019-MCAL-141:switch to cross margin with position and with orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party1 | ETH/FEB23 | cross margin |               |       |

    #AC0019-MCAL-036: When the party switches to cross margin mode, the margin accounts will not be updated until the next MTM
    When the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 26074  | 1476    | 1000 |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 8691        | 9560   | 10429   | 12167   | cross margin | 0             | 0     |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                    |
      | 15800      | TRADING_MODE_MONITORING_AUCTION |

    #AC0019-MCAL-119:switch to isolated margin without position and no orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party3 | ETH/FEB23 | isolated margin | 0.4           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party3 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 0     |

    #AC0019-MCAL-126:increase margin factor without position and no orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party3 | ETH/FEB23 | isolated margin | 0.6           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party3 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.6           | 0     |

    #AC0019-MCAL-127:increase margin factor without position and no orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party3 | ETH/FEB23 | isolated margin | 0.65          |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party3 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.65          | 0     |

    #AC0019-MCAL-135:switch to cross margin without position and no orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party3 | ETH/FEB23 | cross margin | 0.4           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party3 | ETH/FEB23 | 0           | 0      | 0       | 0       | cross margin | 0             | 0     |

    #AC0019-MCAL-125:switch to isolated margin with position and orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.55          |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.55          | 17384 |

    #AC0019-MCAL-133:increase margin factor in isolated margin with position and with orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.56          |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.56          | 17700 |

    #AC0019-MCAL-129:increase margin factor in isolated margin with position and no orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party4 | ETH/FEB23 | isolated margin | 0.5           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party4 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.5           | 0     |
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party4 | ETH/FEB23 | isolated margin | 0.6           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party4 | ETH/FEB23 | 5530        | 0      | 6636    | 0       | isolated margin | 0.6           | 0     |

    #AC0019-MCAL-140:switch to cross margin with position and orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party1 | ETH/FEB23 | cross margin | 0.4           |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 8691        | 9560   | 10429   | 12167   | cross margin | 0             | 0     |

    #AC0019-MCAL-139:switch to cross margin without position and with orders successful in auction
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party5 | ETH/FEB23 | cross margin |               |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party5 | ETH/FEB23 | 1480        | 1628   | 1776    | 2072    | cross margin | 0             | 0     |

