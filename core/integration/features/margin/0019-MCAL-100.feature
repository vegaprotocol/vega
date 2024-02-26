Feature: Test switch between margin mode
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

  Scenario: 001 closeout when party's open position is under maintenance level (0019-MCAL-070)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 172500       |
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | s-1       |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.4           |

    #AC0019-MCAL-100:switch to isolated margin with no position and no order (before the first order ever has been sent) in continuous mode
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/FEB23 | 0      | 171500  |

    When the network moves ahead "2" blocks

    #switch to cross margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode  |
      | party1 | ETH/FEB23 | cross margin |

    #AC0019-MCAL-101:switch back to cross margin with no position and no order in continuous mode
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | cross margin | 0             | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/FEB23 | 0      | 171500  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | buy  | 6      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | b-1       |
    When the network moves ahead "1" blocks

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 9540        | 10494  | 11448   | 13356   | cross margin | 0             | 0     |

    #AC0019-MCAL-106:switch to isolated margin without position and with orders with margin factor such that position margin is < initial should fail in continuous
    #order margin: 6*15800*0.11=10428
    #maintenance margin level in cross margin: 15900*0.1*6=9540
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.11          |       |
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.11          | 10428 |

    And the orders should have the following status:
      | party  | reference | status        |
      | party1 | b-1       | STATUS_ACTIVE |

  Scenario: 002 switch to isolated margin mode without position and no order  (0019-MCAL-110)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 22000        |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | s-1       |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.4           |       |

    #AC0019-MCAL-100:switch to isolated margin with no position and no order (before the first order ever has been sent) in continuous mode
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 21000   | 1000 |

    #AC0019-MCAL-117:update margin factor when already in isolated mode to the same cases as in switch to isolated failures.
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party1 | ETH/FEB23 | isolated margin | 0.4           |       |

    #switch back to cross margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party1 | ETH/FEB23 | cross margin |               |       |
    When the network moves ahead "2" blocks

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | sell | 5      | 15900 | 1                | TYPE_LIMIT | TIF_GTC | s-2       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 7950        | 8745   | 9540    | 11130   | cross margin | 0             | 0     |

    #AC0019-MCAL-115:switch to isolate margin with out of range margin factor
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                                     |
      | party1 | ETH/FEB23 | isolated margin | 0.1           | margin factor (0.1) must be greater than max(riskFactorLong (0.1), riskFactorShort (0.1)) |

    #this number should be validated with correct message
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                      |
      | party1 | ETH/FEB23 | isolated margin | 1.2           | insufficient balance in general account to cover for required order margin |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                                      |
      | party1 | ETH/FEB23 | isolated margin | -0.2          | margin factor (-0.2) must be greater than max(riskFactorLong (0.1), riskFactorShort (0.1)) |

    #AC0019-MCAL-114:switch to isolated margin with position and with orders with margin factor such that there is insufficient balance in the general account in continuous mode
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                                                                      |
      | party1 | ETH/FEB23 | isolated margin | 0.4           | insufficient balance in general account to cover for required order margin |

    #AC0019-MCAL-116:submit update margin mode transaction with no state change (already in cross margin, "change" to cross margin, or already in isolated, submit with same margin factor)
    And the parties submit update margin mode:
      | party  | market    | margin_mode  | margin_factor | error |
      | party1 | ETH/FEB23 | cross margin |               |       |


