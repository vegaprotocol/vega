Bug

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
      | party1           | USD   | 84500        |

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party1 | ETH/FEB23 | 2000              | 0.001 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | party1           | ETH/FEB23 | sell | 8      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 8      |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/FEB23 | 68370       | 75207  | 82044   | 95718   |
    #margin = min(3*(100000-15900), 15900*(0.25))+0.1*15900=5565

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 82044  | 456     | 2000 |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 68370       | 82044   |

    #increase party1's position to 11
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15900      | TRADING_MODE_CONTINUOUS |
    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 72345       | 86814   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 82044  | 456     | 2000 |

    #increase party1's position to 12
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 76320       | 91584   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 84500  | 0       | 0    |
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 6      |

    #increase party1's position to 13
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 80295       | 96354   |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 84500  | 0       | 0    |

    #increase party1's position to 14
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |

    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 84270       | 101124  |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 84500  | 0       | 0    |
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 4      |

    #increase party1's position to 15, and then party's order is cancelled,
    And the parties place the following orders:
      | party           | market id | side | volume | price | resulting trades | type       | tif     |
      | buySideProvider | ETH/FEB23 | buy  | 1      | 15900 | 1                | TYPE_LIMIT | TIF_GTC |
    And the network moves ahead "1" blocks
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party1 | ETH/FEB23 | 83475       | 100170  |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 84500  | 0       | 0    |
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | sell | 15900 | 0      |

    #trigger MTM (should be 1000*15 = 15000) and closeout party1
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
      | party1 | market           | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_SETTLEMENT | ETH/FEB23 | 15000  | USD   |
      | party1 | market           | ACCOUNT_TYPE_MARGIN     | ACCOUNT_TYPE_INSURANCE  | ETH/FEB23 | 69500  | USD   |
      | market | sellSideProvider | ACCOUNT_TYPE_SETTLEMENT | ACCOUNT_TYPE_MARGIN     | ETH/FEB23 | 69500  | USD   |

    And the following trades should be executed:
      | buyer           | price  | size | seller           |
      | buySideProvider | 15900  | 1    | party1           |
      | buySideProvider | 15900  | 1    | party1           |
      | buySideProvider | 16900  | 1    | sellSideProvider |
      | party1          | 16900  | 15   | network          |
      | network         | 200100 | 10   | sellSideProvider |
      | network         | 200000 | 1    | sellSideProvider |


