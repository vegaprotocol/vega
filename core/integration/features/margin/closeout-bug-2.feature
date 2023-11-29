Feature: replicate a closeout bug, when party is distressed, party's order gets cancelled, and then MTM, party gets closed out
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

  Scenario: Check margin and general account when mark price increases and MTM, then closeout (0019-MCAL-070)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 122400       |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party1 | ETH/FEB23 | 2000              | 0.001 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 18     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 18     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 20     | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 100170      | 120204  |
    #margin = 18*min((200000-15900), 15900*(0.25))+18*0.1*15900=100170

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 120204 | 196     | 2000 |

    #trigger MTM (should be 1000*18 = 18000) and closeout party1
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 16900 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 16900 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 0           | 0       |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 0       | 0    |

    And the following transfers should happen:
      | from   | to               | from account            | to account              | market id | amount | asset |
      | party1 | market           | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 18000  | USD   |
      | party1 | market           | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_INSURANCE  | ETH/FEB23 | 10440  | USD   |
      | market | market           | ACCOUNT_TYPE_INSURANCE  | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 10440  | USD   |
      | market | sellSideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/FEB23 | 10440  | USD   |




