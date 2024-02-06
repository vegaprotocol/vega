Feature: Replication a loss socialisation bug
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

  Scenario: 001 closeout when party's open position is under maintenance level
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 95400        |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      # | party1           | ETH/FEB23 | sell | 8      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | s-1       |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    # And the order book should have the following volumes for market "ETH/FEB23":
    #   | side | price  | volume |
    #   | buy  | 14900  | 10     |
    #   | buy  | 15800  | 6      |
    #   | sell | 15900  | 8      |
    #   | sell | 200000 | 1      |
    #   | sell | 200100 | 10     |
    # And the parties should have the following margin levels:
    #   | party  | market id | maintenance | search | initial | release | margin mode  | margin factor | order |
    #   | party1 | ETH/FEB23 | 68370       | 75207  | 82044   | 95718   | cross margin | 0             | 0     |

    # Then the parties should have the following account balances:
    #   | party  | asset | market id | margin | general |
    #   | party1 | USD   | ETH/FEB23 | 82044  | 90456   |

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 55650       | 0      | 66780   | 0       | isolated margin | 0.6           | 0     |

    #position margin: 15900*10*0.6=95400
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 95400  | 0       | 0            |

    #trigger more MTM with party has both short position and short orders
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 17000 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 2      | 17000 | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |
    And the network moves ahead "1" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 17000      | TRADING_MODE_CONTINUOUS |

    #position margin: 15900*10*0.6=95400
    #MTM: 95400-(17000-15900)*10=84400
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 84400  | 0       | 0            |

    And the following transfers should happen:
      | from   | to              | from account            | to account              | market id | amount | asset |
      | party1 | market          | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 11000  | USD   |
      | market | buySideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/FEB23 | 11000  | USD   |

    #trigger more MTM with party has short position
    #MTM: 84400-(25442-17000)*10=0-20
    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider  | ETH/FEB23 | buy  | 1      | 25442 | 0                | TYPE_LIMIT | TIF_GTC |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 25442 | 1                | TYPE_LIMIT | TIF_GTC |
    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 17000      | TRADING_MODE_CONTINUOUS |
    And the network moves ahead "1" blocks

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | order margin |
      | party1 | USD   | ETH/FEB23 | 0      | 0       | 0            |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 25442      | TRADING_MODE_CONTINUOUS |

    And the following transfers should happen:
      | from   | to     | from account        | to account              | market id | amount | asset |
      | party1 | market | ACCOUNT_TYPE_MARGIN | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 84400  | USD   |



