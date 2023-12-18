Feature: Test order amendment which lead to cancellation of all orders and fund returned to the genral account, and active positions shoule be untouched and active
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

  Scenario: 001 when party has open position, check margin and general account when mark price increases and MTM, then closeout (0019-MCAL-070)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 275500       |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 2      | 49920  | 0                | TYPE_LIMIT | TIF_GTC | sell-1    |
      | party1           | ETH/FEB23 | sell | 4      | 49940  | 0                | TYPE_LIMIT | TIF_GTC | sell-2    |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 65190       | 78228   |
    #margin = min(3*(100000-15900), 15900*(0.25))+0.1*15900=5565

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/FEB23 | 78228  | 197272  |

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order  |
      | party1 | ETH/FEB23 | 55650       | 0      | 66780   | 0       | isolated margin | 0.6           | 179760 |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 340     | 179760       |

    #AC 0019-MCAL-068 amend the order (increase size) so that new side margin + margin account balance < maintenance margin, the remainding should be stopped
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error               |
      | party1 | sell-1    | 49940 | 2          | TIF_GTC | margin check failed |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 180100  | 0            |

    And the orders should have the following status:
      | party  | reference | status         |
      | party1 | sell-1    | STATUS_STOPPED |
      | party1 | sell-2    | STATUS_STOPPED |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | sell | 2      | 49920 | 0                | TYPE_LIMIT | TIF_GTC | sell-3    |
      | party1 | ETH/FEB23 | sell | 4      | 49940 | 0                | TYPE_LIMIT | TIF_GTC | sell-4    |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 340     | 179760       |

    #check amendment does happened when new side margin + margin account balance > maintenance margin
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | sell-3    | 49920 | -2         | TIF_GTC |       |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 60244   | 119856       |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | sell-3    | STATUS_CANCELLED |
      | party1 | sell-4    | STATUS_ACTIVE    |

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price  | volume |
      | buy  | 14900  | 10     |
      | buy  | 15800  | 6      |
      | sell | 49920  | 0      |
      | sell | 200000 | 1      |

    #check amendment does happened when new side margin + margin account balance > maintenance margin
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error |
      | party1 | sell-4    | 49910 | 0          | TIF_GTC |       |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 60316   | 119784       |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | sell-3    | STATUS_CANCELLED |
      | party1 | sell-4    | STATUS_ACTIVE    |

    #AC 0019-MCAL-074 amend the order (increase price) so that new side margin + margin account balance < maintenance margin, the remainding should be stopped
    When the parties amend the following orders:
      | party  | reference | price  | size delta | tif     | error               |
      | party1 | sell-4    | 109910 | 0          | TIF_GTC | margin check failed |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 180100  | 0            |

    And the orders should have the following status:
      | party  | reference | status           |
      | party1 | sell-3    | STATUS_CANCELLED |
      | party1 | sell-4    | STATUS_STOPPED   |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | sell | 2      | 49920 | 0                | TYPE_LIMIT | TIF_GTC | sell-5    |
      | party1 | ETH/FEB23 | sell | 4      | 49940 | 0                | TYPE_LIMIT | TIF_GTC | sell-6    |

    #AC 0019-MCAL-075 amend the order (decrease size, increase price) so that new side margin + margin account balance < maintenance margin, the remainding should be stopped
    When the parties amend the following orders:
      | party  | reference | price  | size delta | tif     | error               |
      | party1 | sell-5    | 109910 | -1         | TIF_GTC | margin check failed |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 180100  | 0            |

    And the orders should have the following status:
      | party  | reference | status         |
      | party1 | sell-5    | STATUS_STOPPED |
      | party1 | sell-6    | STATUS_STOPPED |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/FEB23 | sell | 2      | 49920 | 0                | TYPE_LIMIT | TIF_GTC | sell-7    |
      | party1 | ETH/FEB23 | sell | 4      | 49940 | 0                | TYPE_LIMIT | TIF_GTC | sell-8    |

    #AC 0019-MCAL-076 amend the order (increase size, decrease price) so that new side margin + margin account balance < maintenance margin, the remainding should be stopped
    When the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error               |
      | party1 | sell-7    | 19920 | 5          | TIF_GTC | margin check failed |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 180100  | 0            |

    And the orders should have the following status:
      | party  | reference | status         |
      | party1 | sell-7    | STATUS_STOPPED |
      | party1 | sell-8    | STATUS_STOPPED |


