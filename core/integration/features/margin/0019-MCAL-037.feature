Feature: Test magin under isolated margin mode when there is not enough collateral
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

  Scenario: 001 Check margin update when party does not have sufficient collateral
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference  |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |            |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | party-sell |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |            |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |            |

    # Checks for 0019-MCAL-031
    When the network moves ahead "2" blocks
    # Check mark-price matches the specification
    Then the mark price should be "15900" for the market "ETH/FEB23"
    # Check order book matches the specification
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price  | volume |
      | buy  | 14900  | 10     |
      | buy  | 15000  | 1      |
      | sell | 100000 | 1      |
      | sell | 100100 | 10     |
    # Check party margin levels match the specification
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | party | ETH/FEB23 | 9540        | 10494  | 11448   | 13356   |
    #margin = min(3*(100000-15900), 15900*(0.25))+0.1*15900=5565

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 11448  | 36602   |

    #AC: 0019-MCAL-032, switch to isolated margin is rejected becuase selected margin factor is too small
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor | error                                                        |
      | party | ETH/FEB23 | isolated margin | 0.11          | required position margin must be greater than initial margin |

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party | ETH/FEB23 | 9540        | 10494  | 11448   | 13356   | cross margin | 0.9           | 0     |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 11448  | 36602   |

    And the network moves ahead "1" blocks

    #AC: 0019-MCAL-066, 0019-MCAL-037 switch to isolated margin is rejected when party got insufficent balance
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor | error                                                                      |
      | party | ETH/FEB23 | isolated margin | 0.9           | insufficient balance in general account to cover for required order margin |
    And the network moves ahead "2" blocks

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party | ETH/FEB23 | 9540        | 10494  | 11448   | 13356   | cross margin | 0.9           | 0     |

    #AC: 0019-MCAL-066 switch to isolated margin is accepted when party has sufficent balance
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor |
      | party | ETH/FEB23 | isolated margin | 0.5           |
    And the network moves ahead "2" blocks

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 4770        | 0      | 5724    | 0       | isolated margin | 0.5           | 23850 |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 23850  | 350     | 23850        |

    #AC 0019-MCAL-035 order will be rejected if the party does not have enough asset in the general account
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference    | error               |
      | party | ETH/FEB23 | sell | 10     | 15910 | 0                | TYPE_LIMIT | TIF_GTC | sell-order-1 | margin check failed |

    #trigger MTM with party has both short position and short orders
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15890 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15890 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 23880  | 350     | 23850        |

    #trigger more MTM with party has both short position and short orders
    #AC 0019-MCAL-067:When the mark price moves, the margin account should be updated while order margin account should not
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15850 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15850 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 24000  | 350     | 23850        |

    #AC 0019-MCAL-068 amend the order so that new side margin + margin account balance < maintenance margin, the remainding should be stopped
    When the parties amend the following orders:
      | party | reference  | price | size delta | tif     | error               |
      | party | party-sell | 19000 | 0          | TIF_GTC | margin check failed |
    # If the new side margin + margin account balance < maintenance margin =>
    # As the evaluation is the result of any other position/order update, all open orders are stopped and margin re-evaluated.

    And the orders should have the following status:
      | party | reference  | status         |
      | party | party-sell | STATUS_STOPPED |
    And the network moves ahead "1" blocks

    # amend the order which had been stopped
    When the parties amend the following orders:
      | party | reference  | price | size delta | tif     | error                        |
      | party | party-sell | 16500 | 0          | TIF_GTC | OrderError: Invalid Order ID |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 24000  | 24200   | 0            |

    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party | ETH/FEB23 | sell | 2      | 15900 | 0                | TYPE_LIMIT | TIF_GTC | party1-sell-2 |
      | party | ETH/FEB23 | sell | 1      | 15920 | 0                | TYPE_LIMIT | TIF_GTC | party1-sell-3 |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 24000  | 340     | 23860        |

    #AC 0019-MCAL-069 when order is partially filled, the order margin should be udpated
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 31950  | 340     | 15910        |

    And the orders should have the following status:
      | party | reference     | status        |
      | party | party1-sell-2 | STATUS_ACTIVE |
      | party | party1-sell-3 | STATUS_ACTIVE |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 31950  | 340     | 15910        |

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 1      |
      | sell | 15920 | 1      |

  Scenario: 002 replicate panic in testnet
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 3000         |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party            | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC | party-buy |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 3      | 16900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    # Check mark-price matches the specification
    Then the mark price should be "15900" for the market "ETH/FEB23"
    # Check order book matches the specification

    # Check party margin levels match the specification
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | party | ETH/FEB23 | 1590        | 1749   | 1908    | 2226    |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 1800   | 1200    |

    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor | error |
      | party | ETH/FEB23 | isolated margin | 0.2           |       |

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 0           | 0      | 0       |         | isolated margin | 0.2           | 3000  |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 0      | 0       |

    And the network moves ahead "1" blocks

    When the parties amend the following orders:
      | party | reference | price | size delta | tif     | error |
      | party | party-buy | 16900 | 0          | TIF_GTC |       |







