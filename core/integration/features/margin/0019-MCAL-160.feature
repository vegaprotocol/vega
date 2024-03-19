Feature: amend the order and it isn't matched and it's all good and it just moves some funds around and it sits in the book and wait
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

  Scenario: 001 party holds orders only, and party1 holds both orders and positions
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 48050        |
      | party1           | USD   | 48050        |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference   |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party            | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | party-sell  |
      | party1           | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |             |

    When the network moves ahead "2" blocks
    # Check mark-price matches the specification
    Then the mark price should be "15900" for the market "ETH/FEB23"

    #AC: 0019-MCAL-160, 0019-MCAL-161
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error |
      | party  | ETH/FEB23 | isolated margin | 0.2           |       |
      | party1 | ETH/FEB23 | isolated margin | 0.2           |       |

    #maintenance margin: min(3*(15900-15900),15900*3*0.25)+3*0.1*15900=4770
    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 4776        | isolated margin | 0.2           | 9540  |
      | party1 | ETH/FEB23 | 0           | isolated margin | 0.2           | 9540  |

    When the parties amend the following orders:
      | party  | reference   | price | size delta | tif     | error |
      | party  | party-sell  | 16900 | 0          | TIF_GTC |       |
      | party1 | party1-sell | 16900 | 0          | TIF_GTC |       |

    When the network moves ahead "2" blocks

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |
      | sell | 16900 | 6      |

    And the orders should have the following status:
      | party  | reference   | status        |
      | party  | party-sell  | STATUS_ACTIVE |
      | party1 | party1-sell | STATUS_ACTIVE |

    # margin should be:  min(3*(16900-15900),15900*3*0.25)+3*0.1*15900=7770
    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 4776        | isolated margin | 0.2           | 10140 |
      | party1 | ETH/FEB23 | 0           | isolated margin | 0.2           | 10140 |

    #AC: 0019-MCAL-162, 0019-MCAL-163
    When the parties amend the following orders:
      | party  | reference   | price | size delta | tif     | error |
      | party  | party-sell  | 16900 | 1          | TIF_GTC |       |
      | party1 | party1-sell | 16900 | 1          | TIF_GTC |       |
    When the network moves ahead "2" blocks

    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |
      | sell | 16900 | 8      |

    And the orders should have the following status:
      | party  | reference   | status        |
      | party  | party-sell  | STATUS_ACTIVE |
      | party1 | party1-sell | STATUS_ACTIVE |

    #maintenance margin: min(3*(16900-15900),15900*3*0.000125)+3*0.1*15900=4776
    And the parties should have the following margin levels:
      | party  | market id | maintenance | margin mode     | margin factor | order |
      | party  | ETH/FEB23 | 4776        | isolated margin | 0.2           | 13520 |
      | party1 | ETH/FEB23 | 0           | isolated margin | 0.2           | 13520 |

    Then the mark price should be "15900" for the market "ETH/FEB23"
