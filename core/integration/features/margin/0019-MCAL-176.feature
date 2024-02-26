Feature: amend the order and it gets partially matched and you have cover for the trade but not for the remaining order - the trade is done but all of your remaining orders are cancelled
  Background:
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
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.35                   | 0                         | default-futures |

  Scenario: 001 party and party1 both orders and positions
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 113040       |
      | party1           | USD   | 264400       |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 2      | 15300  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party            | ETH/FEB23 | buy  | 6      | 15500  | 0                | TYPE_LIMIT | TIF_GTC | party-buy |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party            | ETH/FEB23 | sell | 6      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 4      | 16900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | buy  | 15500 | 6      |
      | sell | 16900 | 4      |

    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor | error |
      | party | ETH/FEB23 | isolated margin | 0.6           |       |

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 42930       | 0      | 51516   | 0       | isolated margin | 0.6           | 0     |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 57240  | 0       | 55800        |

    #add additional order to reduce exit_price, hence slippage
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 16000 | 0                | TYPE_LIMIT | TIF_GTC | s-liq     |

    #AC: 0019-MCAL-176, amend order price so order get filled/partially filled
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     | error |
      | party | party-buy | 16900 | 0          | TIF_GTC |       |

    And the orders should have the following status:
      | party | reference | status        |
      | party | party-buy | STATUS_FILLED |

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | buy  | 15500 | 0      |

    When the network moves ahead "2" blocks

    #party's order is closed
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.6           | 0     |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general | order margin |
      | party | USD   | ETH/FEB23 | 0      | 113040  | 0            |


