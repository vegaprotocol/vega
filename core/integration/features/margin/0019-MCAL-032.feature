Feature: Test magin under isolated margin mode

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
      | ETH/MAR23 | ETH        | USD   | lqm-params           | simple-risk-model | default-margin-calculator | 1                | default-none | default-none     | default-eth-for-future | 100                    | 0                         | default-futures |


  @SLABug
  Scenario: Check margin update when switch between margin modes (0019-MCAL-031, 0019-MCAL-032)

    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party            | USD   | 100000000000 |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party            | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     |
      | buySideProvider  | ETH/MAR23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/MAR23 | buy  | 1      | 15000  | 0                | TYPE_LIMIT | TIF_GTC |
      | buySideProvider  | ETH/MAR23 | buy  | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | party            | ETH/MAR23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/MAR23 | sell | 1      | 100000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/MAR23 | sell | 10     | 100100 | 0                | TYPE_LIMIT | TIF_GTC |

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
      | party | ETH/FEB23 | 5565        | 6121   | 6678    | 7791    |
    #margin = min((100000-15900), 15900*(0.25))+0.1*15900=5565

    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor |
      | party | ETH/FEB23 | isolated margin | 0.4           |

    And the network moves ahead "2" blocks
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release |
      | party | ETH/FEB23 | 5565        | 6121   | 6678    | 7791    |

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party | ETH/FEB23 | 5565        | 6121   | 6678    | 7791    | cross margin | 0.3           | 0     |

