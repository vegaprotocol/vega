Feature: Test funds are transferred from general account when margin factor decreases and released back to it when it increases
  Background:
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

  Scenario: Parties with same funds, trades, orders and margin factors must have same account balances irrespective of any interim margin factor changes
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 100000       |
      | party2           | USD   | 100000       |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1           | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2           | ETH/FEB23 | sell | 3      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party1           | ETH/FEB23 | sell | 3      | 16100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2           | ETH/FEB23 | sell | 3      | 16100  | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |

    When the network moves ahead "2" blocks
    Then the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900        | TRADING_MODE_CONTINUOUS |

    When the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.5           |
      | party2 | ETH/FEB23 | isolated margin | 0.5           |
    And the network moves ahead "2" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5370        | 0      | 6444    | 0       | isolated margin | 0.5           | 24150 |
      | party2 | ETH/FEB23 | 5370        | 0      | 6444    | 0       | isolated margin | 0.5           | 24150 |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 23850  | 52000   | 24150        |
      | party2 | USD   | ETH/FEB23 | 23850  | 52000   | 24150        |

    When the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party2 | ETH/FEB23 | isolated margin | 0.3           |
    And the network moves ahead "2" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5370        | 0      | 6444    | 0       | isolated margin | 0.5           | 24150 |
      | party2 | ETH/FEB23 | 5370        | 0      | 6444    | 0       | isolated margin | 0.3           | 14490 |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 23850  | 52000   | 24150        |
      | party2 | USD   | ETH/FEB23 | 14310  | 71200   | 14490        |

    When the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party2 | ETH/FEB23 | isolated margin | 0.5           |
    And the network moves ahead "2" blocks
    # Expecting equal margin levels and balances at this stage
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 5370        | 0      | 6444    | 0       | isolated margin | 0.5           | 24150 |
      | party2 | ETH/FEB23 | 5370        | 0      | 6444    | 0       | isolated margin | 0.5           | 24150 |
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 23850  | 52000   | 24150        |
      | party2 | USD   | ETH/FEB23 | 23850  | 52000   | 24150        |