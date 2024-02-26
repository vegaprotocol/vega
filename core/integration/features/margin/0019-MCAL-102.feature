Feature: Switch mode during auction
  Background:
    # switch to isolated margin with no position and no order (before the first order ever has been sent) in auction
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

    And the following network parameters are set:
      | name                           | value |
      | market.auction.minimumDuration | 2     |

  Scenario: 001 switch to isolated margin with no position and no order (before the first order ever has been sent) in auction (0019-MCAL-102)
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 273500       |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 6      | 15800  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    And the order book should have the following volumes for market "ETH/FEB23":
      | side | price  | volume |
      | buy  | 14900  | 10     |
      | buy  | 15800  | 6      |
      | sell | 15900  | 0      |
      | sell | 200000 | 1      |
    And the parties should have the following margin levels:
      | party           | market id | maintenance |
      | buySideProvider | ETH/FEB23 | 24380       |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 272500  | 1000 |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 |
      | 0          | TRADING_MODE_OPENING_AUCTION |

    #switch to isolated margin, failed because party has no order
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                      |
      | party1 | ETH/FEB23 | isolated margin | 0.6           | no market observable price |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/FEB23 | sell | 8      | 15900 | 0                | TYPE_LIMIT | TIF_GTC | party1-sell |

    And the orders should have the following status:
      | party  | reference   | status        |
      | party1 | party1-sell | STATUS_ACTIVE |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 |
      | 0          | TRADING_MODE_OPENING_AUCTION |
    When the network moves ahead "1" blocks

    #switch to isolated margin, failed because of no market observable price
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor | error                      |
      | party1 | ETH/FEB23 | isolated margin | 0.6           | no market observable price |

    When the network moves ahead "1" blocks
    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 12720       | 15264   | cross margin | 0             | 0     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 15264  | 257236  | 1000 |

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode                 |
      | 0          | TRADING_MODE_OPENING_AUCTION |

    And the parties place the following orders:
      | party            | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15800 | 0                | TYPE_LIMIT | TIF_GTC | sellP-2   |

    When the network moves ahead "2" blocks

    And the market data for the market "ETH/FEB23" should be:
      | mark price | trading mode            |
      | 15800      | TRADING_MODE_CONTINUOUS |
    #MTM from price change
    And the parties should have the following margin levels:
      | party  | market id | maintenance | initial | search | release | margin mode  | margin factor | order |
      | party1 | ETH/FEB23 | 12640       | 15168   | 13904  | 17696   | cross margin | 0             | 0     |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 15264  | 257236  | 1000 |

    #switch to isolated margin, when there is market observable price
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.6           |
    When the network moves ahead "1" blocks
    And the parties should have the following margin levels:
      | party  | market id | maintenance | release | search | initial | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0       | 0      | 0       | isolated margin | 0.6           | 76320 |
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party1 | USD   | ETH/FEB23 | 0      | 196180  | 1000 |


