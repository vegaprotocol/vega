Feature: Futures market can be created with a with [hardcoded risk factors](./0018-RSKM-quant_risk_models.ipynb).
  Background:
    # Set liquidity parameters to allow "zero" target-stake which is needed to construct the order-book defined in the ACs
    Given the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 1s    |
    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.001            | 24h         | 1e-9           |
    And the simple risk model named "simple-risk-model":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.2   | 100         | -100          | 0.2                    |
    And the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.0002        | 0.01 | 0  | 0.0 | 1.2   |
    #rf_long: 0.369668054
    #rf_short: 0.5650462
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model        | margin calculator         | auction duration | fees         | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params      | max price cap | fully collateralised | binary |
      | ETH/FEB23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 0.25                   | 0                         | default-futures | 19000         | false                | false  |
  @NoPerp
  Scenario: 001 0016-PFUT-026, 0016-PFUT-028
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount  |
      | buySideProvider  | USD   | 1000000 |
      | sellSideProvider | USD   | 1000000 |
      | aux1             | USD   | 1000000 |
      | aux2             | USD   | 100000  |
      | party            | USD   | 480500  |
      | party1           | USD   | 480500  |
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | aux1             | ETH/FEB23 | buy  | 10     | 14900 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | buySideProvider  | ETH/FEB23 | buy  | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party            | ETH/FEB23 | sell | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | party            | ETH/FEB23 | sell | 3      | 15900 | 0                | TYPE_LIMIT | TIF_GTC | party-sell  |
      | party1           | ETH/FEB23 | sell | 3      | 16100 | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 18100 | 0                | TYPE_LIMIT | TIF_GTC |             |
      | aux2             | ETH/FEB23 | sell | 10     | 18200 | 0                | TYPE_LIMIT | TIF_GTC |             |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    And the average fill price is:
      | market    | volume | side | ref price | mark price | equivalent linear slippage factor |
      | ETH/FEB23 | 3      | sell | 15900     | 15900      | 0                                 |

    #party margin:15900*(0.25+0.2)*3 +15900*0.2*3=31005
    And the parties should have the following margin levels:
      | party | market id | maintenance |
      | party | ETH/FEB23 | 31005       |
      | aux1  | ETH/FEB23 | 15900       |
      | aux2  | ETH/FEB23 | 31800       |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 37206  | 443294  |
      | aux1  | USD   | ETH/FEB23 | 17880  | 982120  |
      | aux2  | USD   | ETH/FEB23 | 43680  | 56320   |

    #0016-PFUT-028: Updating a risk model on a futures market with [hardcoded risk factors]
    And the markets are updated:
      | id        | risk model             |
      | ETH/FEB23 | lognormal-risk-model-1 |

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |           |
    And the network moves ahead "3" blocks

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 83767  | 396733  |
      | aux1  | USD   | ETH/FEB23 | 70533  | 929467  |
      | aux2  | USD   | ETH/FEB23 | 100000 | 0       |

    #party margin:15900*(0.25+0.5650462)*4 +15900*0.5650462*2=69806
    And the parties should have the following margin levels:
      | party | market id | maintenance |
      | party | ETH/FEB23 | 69806       |
      | aux1  | ETH/FEB23 | 58778       |
      | aux2  | ETH/FEB23 | 89843       |

    #0016-PFUT-027: Updating a risk model on a futures market with regular risk model to with [hardcoded risk factors]
    And the markets are updated:
      | id        | risk model        |
      | ETH/FEB23 | simple-risk-model |

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |           |
    And the network moves ahead "3" blocks

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general |
      | party | USD   | ETH/FEB23 | 46746  | 433754  |
      | aux1  | USD   | ETH/FEB23 | 19080  | 980920  |
      | aux2  | USD   | ETH/FEB23 | 38160  | 61840   |

    #party margin:15900*(0.25+0.2)*5 +15900*0.2*1=69806=38955
    And the parties should have the following margin levels:
      | party | market id | maintenance |
      | party | ETH/FEB23 | 38955       |
      | aux1  | ETH/FEB23 | 15900       |
      | aux2  | ETH/FEB23 | 31800       |
