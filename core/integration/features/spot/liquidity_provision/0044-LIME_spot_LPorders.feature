Feature: Spot market

  Scenario: party submit liquidity, and amend/cancel it

  Background:

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 300               |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | market.liquidityV2.earlyExitPenalty                 | 0.02  |
      | market.stake.target.timeWindow                      | 2s    |
      | market.liquidity.providers.fee.distributionTimeStep | 0     |

    Given time is updated to "2023-07-20T00:00:00Z"
    Given the average block duration is "2"

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params    |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | default-basic |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 50     |
      | lpprov | ETH   | 2000   |
      | lpprov | BTC   | 1000   |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.2            |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | lp type    |
      | lp1 | lpprov | BTC/ETH   | 1000              | 0   | submission |

    Then the liquidity provisions should have the following states:
      | id  | party  | market  | commitment amount | status        |
      | lp1 | lpprov | BTC/ETH | 1000              | STATUS_ACTIVE |

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH   | buy  | 1      | 12    | 0                | TYPE_LIMIT | TIF_GTC | party-order1 |      |
      | party2 | BTC/ETH   | sell | 1      | 19    | 0                | TYPE_LIMIT | TIF_GTC | party-order2 |      |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | lpprov | BTC/ETH   | buy  | 10     | 10    | 0                | TYPE_LIMIT | TIF_GTC | lp-order1    |      |
      | lpprov | BTC/ETH   | sell | 5      | 20    | 0                | TYPE_LIMIT | TIF_GTC | lp-order2    |      |

    Then the network moves ahead "10" blocks

    When the opening auction period ends for market "BTC/ETH"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 360000  | 10        | 22        | 200          | 1000           | 0             |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 12    | 1      |
      | buy  | 10    | 10     |
      | sell | 19    | 1      |
      | sell | 20    | 5      |

    Then "lpprov" should have holding account balance of "1000" for asset "ETH"
    Then "lpprov" should have general account balance of "0" for asset "ETH"
    Then "lpprov" should have holding account balance of "50" for asset "BTC"
    Then "lpprov" should have general account balance of "950" for asset "BTC"
    Then the party "lpprov" lp liquidity bond account balance should be "1000" for the market "BTC/ETH"

    Then the liquidity provisions should have the following states:
      | id  | party  | market  | commitment amount | status        |
      | lp1 | lpprov | BTC/ETH | 1000              | STATUS_ACTIVE |

    Then the parties amend the following orders:
      | party  | reference | price | size delta | tif     | error                                                              |
      | lpprov | lp-order1 | 10    | 90         | TIF_GTC | party does not have sufficient balance to cover the trade and fees |
      | lpprov | lp-order2 | 20    | 45         | TIF_GTC |                                                                    |

    And the orders should have the following status:
      | party  | reference | status        |
      | lpprov | lp-order1 | STATUS_ACTIVE |
      | lpprov | lp-order2 | STATUS_ACTIVE |

    And the order book should have the following volumes for market "BTC/ETH":
      | side | price | volume |
      | buy  | 12    | 1      |
      | buy  | 10    | 10     |
      | sell | 19    | 1      |
      | sell | 20    | 50     |

    Then "lpprov" should have holding account balance of "1000" for asset "ETH"
    Then "lpprov" should have general account balance of "0" for asset "ETH"
    Then "lpprov" should have holding account balance of "500" for asset "BTC"
    Then "lpprov" should have general account balance of "500" for asset "BTC"
