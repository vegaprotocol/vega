Feature: Iceberg orders

  Background:

    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |
    #calculated risk factor long: 0.336895684; risk factor short: 0.4878731

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 100     | 0.99999999  | 300               |
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 2              |

    And the oracle spec for settlement data filtering data from "0xCAFECAFE19" named "ethDec19Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |

    And the oracle spec for trading termination filtering data from "0xCAFECAFE19" named "ethDec19Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |
    Given the markets:
      | id        | quote name | asset | risk model             | margin calculator   | auction duration | fees         | price monitoring   | data source config | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/DEC19 | ETH        | USD   | lognormal-risk-model-1 | margin-calculator-1 | 1                | default-none | price-monitoring-1 | ethDec19Oracle     | 1e6                    | 1e6                       | default-futures |
    And the following network parameters are set:
      | name                                    | value |
      | market.auction.minimumDuration          | 1     |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 1500  |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @batch
  Scenario: Batch with normal orders and icebergs, 0014-ORDT-014, 0014-ORDT-015
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party1 | USD   | 10000      |
      | party2 | USD   | 10000      |
      | party3 | USD   | 1000000000 |
      | party4 | USD | 1000000 |
      | party5 | USD | 1000000 |
      | aux    | USD   | 1000000    |
      | aux2   | USD   | 100000     |
      | lpprov | USD   | 90000000   |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
      | lp1 | lpprov | ETH/DEC19 | 90000000          | 0.1 | submission |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume     | offset |
      | lpprov | ETH/DEC19 | 2         | 1                    | buy  | BID              | 50         | 100    |
      | lpprov | ETH/DEC19 | 2         | 1                    | sell | ASK              | 50         | 100    |

    # place auxiliary orders so we always have best bid and best offer as to not trigger the liquidity auction
    When the parties place the following orders:
      | party | market id | side | volume | price | resulting trades | type       | tif     |
      | aux   | ETH/DEC19 | buy  | 1      | 99    | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 101   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux2  | ETH/DEC19 | buy  | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
      | aux   | ETH/DEC19 | sell | 1      | 100   | 0                | TYPE_LIMIT | TIF_GTC |
    Then the opening auction period ends for market "ETH/DEC19"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "100" for the market "ETH/DEC19"

    And the parties place the following iceberg orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | peak size | minimum visible size |
      | party1 | ETH/DEC19 | sell | 6      | 100   | 0                | TYPE_LIMIT | TIF_GTC | this-order-1 | 4         | 5                    |
      | party2 | ETH/DEC19 | sell | 3      | 100   | 0                | TYPE_LIMIT | TIF_GTC | this-order-2 | 4         | 5                    |
      | party1 | ETH/DEC19 | sell | 100    | 101   | 0                | TYPE_LIMIT | TIF_GTC | this-order-3 | 4         | 5                    |
      | party2 | ETH/DEC19 | buy  | 100    | 99    | 0                | TYPE_LIMIT | TIF_GTC | this-order-4 | 4         | 5                    |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party1 | USD   | ETH/DEC19 | 7758   | 2242    |
      | party2 | USD   | ETH/DEC19 | 5053   | 4947    |

# margin initial = 0.4878731*100*2*1.5 = 147

    Then the party "party3" starts a batch instruction

    Then the party "party3" adds the following orders to a batch:
      | market id | side | volume | price | type       | tif     | reference |
      | ETH/DEC19 | buy  | 4      | 101   | TYPE_LIMIT | TIF_GTC | party3    |

    Then the party "party3" adds the following iceberg orders to a batch:
      | market id | side | volume | price | type       | tif     | reference    | peak size | minimum visible size |
      | ETH/DEC19 | buy  | 3      | 101   | TYPE_LIMIT | TIF_GTC | this-order-5 | 2         | 1                    |
      | ETH/DEC19 | buy  | 4      | 101   | TYPE_LIMIT | TIF_GTC | this-order-6 | 2         | 1                    |

    Then the party "party3" submits their batch instruction

#Iceberg order trading during continous mode
    Then the following trades should be executed:
      | buyer  | seller | price | size |
      | party3 | party1 | 100   | 4    |
      | party3 | party2 | 100   | 3    |
      | party3 | party1 | 100   | 2    |

    And the network moves ahead "10" blocks

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party1 | USD   | ETH/DEC19 | 7752   | 2242      |
      | party2 | USD   | ETH/DEC19 | 5050   | 4947      |
      | party3 | USD   | ETH/DEC19 | 576    | 999999321 |


    #cancel an order which had been traded
    When the parties cancel the following orders:
      | party  | reference    | error                                  |
      | party3 | this-order-5 | unable to find the order in the market |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC19"
    And the mark price should be "101" for the market "ETH/DEC19"

    Then the party "party4" starts a batch instruction

    Then the party "party4" adds the following iceberg orders to a batch:
      | market id | side | volume | price | type       | tif     | reference    | peak size | minimum visible size |
      | ETH/DEC19 | buy | 6 | 100 | TYPE_LIMIT | TIF_GTC | this-order-7 | 3 | 1 |
      | ETH/DEC19 | buy  | 8      | 100   | TYPE_LIMIT | TIF_GTC | this-order-8 | 2         | 1                    |
    Then the party "party4" adds the following orders to a batch:
      | market id | side | volume | price | type       | tif     | reference           |
      | ETH/DEC19 | buy | 8 | 100 | TYPE_LIMIT | TIF_GTC | party4-normal-order |

    Then the party "party4" submits their batch instruction

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party4 | USD | ETH/DEC19 | 1123 | 998877 |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party4 | ETH/DEC19 | 749         | 898    | 1123    | 1498    |

    And the network moves ahead "10" blocks

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party4 | USD   | ETH/DEC19 | 1123   | 998877  |

    When the parties cancel the following orders:
      | party  | reference           |
      | party4 | party4-normal-order |
      | party4 | this-order-7 |

#iceberg order canceled
    Then the iceberg orders should have the following states:
      | party  | market id | side | visible volume | price | status           | reserved volume | reference    |
      | party4 | ETH/DEC19 | buy  | 3              | 100   | STATUS_CANCELLED | 3               | this-order-7 |

    And the network moves ahead "10" blocks
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party4 | USD   | ETH/DEC19 | 1123   | 998877  |
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party4 | ETH/DEC19 | 749         | 898    | 1123    | 1498    |

    When the parties cancel the following orders:
      | party  | reference    |
      | party4 | this-order-8 |

#margin released after order cancellation
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general |
      | party4 | USD   | ETH/DEC19 | 0      | 1000000 |
