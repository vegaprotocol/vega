Feature: when party holds both orders and positions, amend order so order is filled\partially filled while party does not have enough collateral to cover
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
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures |

  Scenario: 001 party and party1 both orders and positions
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 26440        |
      | party1           | USD   | 26440        |
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

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |
      | sell | 16900 | 10     |

    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party  | ETH/FEB23 | isolated margin | 0.2           |       |
      | party1 | ETH/FEB23 | isolated margin | 0.2           |       |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 7770        | 0      | 9324    | 0       | isolated margin | 0.2           | 16900 |
      | party1 | ETH/FEB23 | 7770        | 0      | 9324    | 0       | isolated margin | 0.2           | 16900 |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party  | USD   | ETH/FEB23 | 9540   | 0       | 16900        |
      | party1 | USD   | ETH/FEB23 | 9540   | 0       | 16900        |

    #AC: 0019-MCAL-168, 0019-MCAL-169, amend order price so order get filled/partially filled
    When the parties amend the following orders:
      | party  | reference   | price | size delta | tif     | error |
      | party  | party-sell  | 15500 | 0          | TIF_GTC |       |
      | party1 | party1-sell | 15300 | 0          | TIF_GTC |       |

    Then the orders should have the following status:
      | party  | reference   | status        |
      | party  | party-sell  | STATUS_FILLED |
      | party1 | party1-sell | STATUS_ACTIVE |

    #margin for party: 15900*0.2*3+15500*0.2*5=25040
    #margin for party1: 15900*0.2*3+15300*0.2*2=15660
    #order margin for party1: 15300*0.2*3=9180

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party  | USD   | ETH/FEB23 | 25040  | 1400    | 0            |
      | party1 | USD   | ETH/FEB23 | 15660  | 1600    | 9180         |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 44520       | 0      | 53424   | 0       | isolated margin | 0.2           | 0     |
      | party1 | ETH/FEB23 | 27825       | 0      | 33390   | 0       | isolated margin | 0.2           | 9180  |

    #party1's order is partially filled, and the rest of the order is left on the order book
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15300 | 3      |
    When the network moves ahead "2" blocks
    Then the mark price should be "15300" for the market "ETH/FEB23"

    Then the orders should have the following status:
      | party  | reference   | status        |
      | party  | party-sell  | STATUS_FILLED |
      | party1 | party1-sell | STATUS_FILLED |

    #party1's order is partially filled, and the rest of the order is canceled
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15300 | 0      |

    # in M2M party has insufficient margin hence getting closed out, party1 then trades with the network...
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.2           | 0     |
      | party1 | ETH/FEB23 | 16065       | 0      | 19278   | 0       | isolated margin | 0.2           | 0     |

    #margin for party1: 3*0.2*15300=9180
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party  | USD   | ETH/FEB23 | 0      | 1400    | 0            |
      | party1 | USD   | ETH/FEB23 | 9180   | 1600    | 0            |

    And the following trades should be executed:
      | buyer           | price | size | seller  |
      | buySideProvider | 15900 | 3    | party   |
      | buySideProvider | 15900 | 3    | party1  |
      | buySideProvider | 15500 | 5    | party   |
      | buySideProvider | 15300 | 2    | party1  |
      | party           | 15300 | 8    | network |
      | party1          | 15300 | 5    | network |
      | network         | 15300 | 3    | party1  |
