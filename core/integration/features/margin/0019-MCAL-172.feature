Feature: when party holds both orders and positions, amend order so order is filled\partially filled while party does have enough collateral to cover
  Background:
    # Set liquidity parameters to allow "zero" target-stake which is needed to construct the order-book defined in the ACs
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.00             | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 100         | -100          | 0.2                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.000125               | 0                         | default-futures |

  Scenario: 001 party and party1 both orders and positions
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 12644000     |
      | party1           | USD   | 12644000     |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference   |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 2      | 15300  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 5      | 15500  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party1           | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party            | ETH/FEB23 | sell | 5      | 16900  | 0                | TYPE_LIMIT | TIF_GTC | party-sell  |
      | party1           | ETH/FEB23 | sell | 5      | 16900  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |             |
    # update linear slippage factor more in line with what book-based slippage used to be
    And the markets are updated:
      | id        | linear slippage factor |
      | ETH/FEB23 | 0.0628930817610063     |
    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |
      | sell | 16900 | 10     |

    #0.1628930817610063*3*15900=7770
    #0.100125*3*15900=4775
    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode  | margin factor | order |
      | party  | ETH/FEB23 | 15721       | cross margin | 0             | 0     |
      | party1 | ETH/FEB23 | 15721       | cross margin | 0             | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general  |
      | party  | USD   | ETH/FEB23 | 18865  | 12625135 |
      | party1 | USD   | ETH/FEB23 | 18865  | 12625135 |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party  | ETH/FEB23 | isolated margin | 0.2           |       |
      | party1 | ETH/FEB23 | isolated margin | 0.2           |       |
    And the average fill price is:
      | market    | volume | side | ref price | mark price | equivalent linear slippage factor |
      | ETH/FEB23 | 5      | sell | 15900     | 15900      | 0.0628930817610063                |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 7771        | isolated margin | 0.2           | 16900 |
      | party1 | ETH/FEB23 | 7771        | isolated margin | 0.2           | 16900 |


    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general  | order margin |
      | party  | USD   | ETH/FEB23 | 9540   | 12617560 | 16900        |
      | party1 | USD   | ETH/FEB23 | 9540   | 12617560 | 16900        |

    #add additional order to reduce exit_price, hence slippage
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 16000 | 0                | TYPE_LIMIT | TIF_GTC | s-liq     |

    #AC: 0019-MCAL-172, 0019-MCAL-173, amend order price so order get filled/partially filled
    When the parties amend the following orders:
      | party  | reference   | price | size delta | tif     | error |
      | party  | party-sell  | 15500 | 0          | TIF_GTC |       |
      | party1 | party1-sell | 15300 | 0          | TIF_GTC |       |

    And the orders should have the following status:
      | party  | reference   | status        |
      | party  | party-sell  | STATUS_FILLED |
      | party1 | party1-sell | STATUS_ACTIVE |

    And the markets are updated:
      | id        | linear slippage factor |
      | ETH/FEB23 | 0.05                   |

    When the network moves ahead "2" blocks

    #0.15*8*15500

    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 18360       | isolated margin | 0.2           | 0     |
      | party1 | ETH/FEB23 | 11475       | isolated margin | 0.2           | 9180  |

    And the markets are updated:
      | id        | linear slippage factor |
      | ETH/FEB23 | 0.35                   |

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | buySideProvider | ETH/FEB23 | buy  | 5      | 15300 | 1                | TYPE_LIMIT | TIF_GTC |              |
      | buySideProvider | ETH/FEB23 | buy  | 5      | 15500 | 0                | TYPE_LIMIT | TIF_GTC |              |
      | party           | ETH/FEB23 | sell | 5      | 16900 | 0                | TYPE_LIMIT | TIF_GTC | party-sell2  |
      | party1          | ETH/FEB23 | sell | 3      | 16900 | 0                | TYPE_LIMIT | TIF_GTC | party1-sell2 |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 55080       | isolated margin | 0.2           | 16900 |
      | party1 | ETH/FEB23 | 55080       | isolated margin | 0.2           | 10140 |

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |
      | sell | 16900 | 8      |

    #AC: 0019-MCAL-174, 0019-MCAL-175, amend order price so order get filled/partially filled
    #however, after trade, their margin balance will be less than the margin maintenance level, so order will be stopped
    When the parties amend the following orders:
      | party  | reference    | price | size delta | tif     | error               |
      | party  | party-sell2  | 15500 | 0          | TIF_GTC | margin check failed |
      | party1 | party1-sell2 | 15300 | 0          | TIF_GTC | margin check failed |

    And the orders should have the following status:
      | party  | reference    | status         |
      | party  | party-sell2  | STATUS_STOPPED |
      | party1 | party1-sell2 | STATUS_STOPPED |





