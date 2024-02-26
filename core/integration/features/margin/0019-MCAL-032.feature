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

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 6678   | 99999890494 |

    #AC: 0019-MCAL-032, switch to isolated margin is rejected becuase selected margin factor is too small
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor | error                                                        |
      | party | ETH/FEB23 | isolated margin | 0.11          | required position margin must be greater than initial margin |

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party | ETH/FEB23 | 5565        | 6121   | 6678    | 7791    | cross margin | 0             | 0     |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 6678   | 99999890494 |

    And the network moves ahead "1" blocks
    #AC: 0019-MCAL-033, switch to isolated margin
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor |
      | party | ETH/FEB23 | isolated margin | 0.9           |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 14310  | 99999882862 |
    And the network moves ahead "2" blocks

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 5565        | 0      | 6678    | 0       | isolated margin | 0.9           | 0     |

    #AC: 0019-MCAL-031, decrease margin factor
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor | error |
      | party | ETH/FEB23 | isolated margin | 0.7           |       |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 11130  | 99999886042 |
    And the network moves ahead "2" blocks

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 5565        | 0      | 6678    | 0       | isolated margin | 0.7           | 0     |

    #AC: 0019-MCAL-059, increase margin factor
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor |
      | party | ETH/FEB23 | isolated margin | 0.9           |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 14310  | 99999882862 |
    And the network moves ahead "2" blocks

    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party | ETH/FEB23 | 5565        | 0      | 6678    | 0       | isolated margin | 0.9           | 0     |

    #AC: 0019-MCAL-065, switch margin mode from isolated margin to cross margin when party holds position only
    And the parties submit update margin mode:
      | party | market    | margin_mode  | margin_factor |
      | party | ETH/FEB23 | cross margin |               |
    And the network moves ahead "1" blocks
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
      | party | ETH/FEB23 | 5565        | 6121   | 6678    | 7791    | cross margin | 0             | 0     |
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 14310  | 99999882862 |

    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor |
      | party | ETH/FEB23 | isolated margin | 0.9           |
    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 14310  | 99999882862 |

    #trigger MTM
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 15910 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15910 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 14310  | 99999882862 |

    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     |
      | party | USD   | ETH/FEB23 | 14300  | 99999882862 |

    #AC: 0019-MCAL-034, party places a new order which can not offset their position
    #addional margin should be: limit price x current position x new margin factor = 15910 x 10 x 0.9 = 143190
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party | ETH/FEB23 | sell | 10     | 15912 | 0                | TYPE_LIMIT | TIF_GTC | sell-10   |
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 14300  | 99999739654 | 143208       |

    #AC 0019-MCAL-060, Amend order,check the updated order margin
    When the parties amend the following orders:
      | party | reference | price | size delta | tif     |
      | party | sell-10   | 15912 | -5         | TIF_GTC |
    # And the network moves ahead "1" blocks

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 14300  | 99999811258 | 71604        |

    #AC 0019-MCAL-061, party's order get partially filled, check the updated margin account and order account
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 3      | 15912 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 57262  | 99999811258 | 28642        |

    Then the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15912 | 2      |

    #AC: 0019-MCAL-063, switch margin mode from isolated margin to cross margin when party holds both position and orders
    And the parties submit update margin mode:
      | party | market    | margin_mode  |
      | party | ETH/FEB23 | cross margin |

    And the network moves ahead "1" blocks
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode  | order | margin factor |
      | party | ETH/FEB23 | 25460       | 28006  | 30552   | 35644   | cross margin | 0     | 0             |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 30552  | 99999866608 | 0            |

    #AC: 0019-MCAL-064, switch margin mode from cross margin to isolated margin when party holds both position and orders
    And the parties submit update margin mode:
      | party | market    | margin_mode     | margin_factor |
      | party | ETH/FEB23 | isolated margin | 0.9           |
    And the network moves ahead "1" blocks
    And the parties should have the following margin levels:
      | party | market id | maintenance | search | initial | release | margin mode     | order | margin factor |
      | party | ETH/FEB23 | 22277       | 0      | 26732   | 0       | isolated margin | 28641 | 0.9           |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 57272  | 99999811247 | 28641        |

    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15912 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 71592  | 99999811247 | 14321        |

    #AC 0019-MCAL-062, when party has no orders, the order margin account shoule be 0
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15912 | 1                | TYPE_LIMIT | TIF_GTC |

    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 85912  | 99999811248 | 0            |

    #AC: 0019-MCAL-038,when party places a new order which can offset the party's position, no additional margin will be needed
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party | ETH/FEB23 | buy  | 3      | 15912 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 85912  | 99999811248 | 0            |

    #AC: 0019-MCAL-039,when party places a large order which can offset all of the party's position and then add new orders, additional margin will be needed
    And the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | party | ETH/FEB23 | buy  | 10     | 15912 | 0                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the parties should have the following account balances:
      | party | asset | market id | margin | general     | order margin |
      | party | USD   | ETH/FEB23 | 85912  | 99999711003 | 100245       |




