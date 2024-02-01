Feature: Test switch between margin mode
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

  Scenario: Amending an order to make it match while in isolated mode can fail if the margin requirements are not met.
    Given the parties deposit on asset's general account the following amount:
      | party            | asset | amount       |
      | buySideProvider  | USD   | 100000000000 |
      | sellSideProvider | USD   | 100000000000 |
      | party1           | USD   | 16000        |
    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | party1 | ETH/FEB23 | 1000              | 0.1 | submission |

    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 14900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | buy  | 10     | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | buySideProvider  | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 15900  | 0                | TYPE_LIMIT | TIF_GTC | s-1       |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 200000 | 0                | TYPE_LIMIT | TIF_GTC |           |
      | sellSideProvider | ETH/FEB23 | sell | 10     | 200100 | 0                | TYPE_LIMIT | TIF_GTC |           |

    When the network moves ahead "2" blocks
    Then the mark price should be "15900" for the market "ETH/FEB23"

    #switch to isolated margin
    And the parties submit update margin mode:
      | party  | market    | margin_mode     | margin_factor |
      | party1 | ETH/FEB23 | isolated margin | 0.4           |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 0     |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/FEB23 | 0      | 15000   |

    # Now place an order that sits on the book which we can amend to match later
    And the parties place the following orders:
      | party            | market id | side | volume | price  | resulting trades | type       | tif     | reference  |
      | sellSideProvider | ETH/FEB23 | sell | 1      | 20000  | 0                | TYPE_LIMIT | TIF_GTC | sell-1     |
      | party1           | ETH/FEB23 | buy  | 1      | 19000  | 0                | TYPE_LIMIT | TIF_GTC | party1-buy |

    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 7600  |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/FEB23 | 0      | 7400    |

    # Set up two orders which do not cross so we can amend one of them to force the trade
    Then the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | buy  | 19000 | 1      |
      | sell | 20000 | 1      |

    # This will make the two orders cross but will fail the margin check once the order has already been cancelled/replaced.
    When the parties amend the following orders:
      | party | reference  | price | tif     | error               |
      | party1| party1-buy | 20001 | TIF_GTC | margin check failed |

    # party 1 no longer has it's order due to insufficient margin
    And the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release | margin mode     | margin factor | order |
      | party1 | ETH/FEB23 | 0           | 0      | 0       | 0       | isolated margin | 0.4           | 0     |

    Then the orders should have the following status:
      | party            | reference   | status         |
      | party1           | party1-buy  | STATUS_STOPPED |
      | sellSideProvider | sell-1      | STATUS_ACTIVE  |

    # The amended order should be removed from the book and the order it was trying to match with will be restored
    Then the order book should have the following volumes for market "ETH/FEB23":
      | side | price | volume |
      | buy  | 19000 | 0      |
      | sell | 20000 | 1      |


