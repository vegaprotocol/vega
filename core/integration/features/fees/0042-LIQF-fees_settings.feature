Feature: Test liquidity fee settings, using 3 different methods

  Background:

    Given the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 500         | 500           | 0.1                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_CONSTANT      | 0.08                   |
    And the fees configuration named "fees-config-2":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_CONSTANT      | 0.01                   |
    And the fees configuration named "fees-config-3":
      | maker fee | infrastructure fee | liquidity fee method    | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_WEIGHTED_AVERAGE | 0.01                   |
    And the fees configuration named "fees-config-4":
      | maker fee | infrastructure fee | liquidity fee method | liquidity fee constant |
      | 0.0004    | 0.001              | METHOD_MARGINAL_COST | 0.01                   |

    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |
    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.0              | 24h         | 1.0            |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1.0         | 0.5                          | 1                             | 1.0                    |
    And the following network parameters are set:
      | name                                    | value |
      | market.value.windowLength               | 1h    |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 12    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model          | margin calculator         | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR22 | USD        | USD   | lqm-params           | simple-risk-model-1 | default-margin-calculator | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 0.2                    | 0                         | SLA        |

  Scenario: 001 Liquidity fee setting to METHOD_CONSTANT(0042-LIQF-058, 0042-LIQF-061), METHOD_MARGINAL_COST(0042-LIQF-062), and METHOD_WEIGHTED_AVERAGE(0042-LIQF-057, 0042-LIQF-056)
    # setup accounts
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | lp1    | USD   | 1000000000 |
      | lp2    | USD   | 1000000000 |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |

    And the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type    |
      | lp1 | lp1   | ETH/MAR22 | 800               | 0.005 | submission |
      | lp2 | lp2   | ETH/MAR22 | 300               | 0.004 | submission |

    And the parties place the following pegged iceberg orders:
      | party | market id | peak size | minimum visible size | side | pegged reference | volume | offset |
      | lp1   | ETH/MAR22 | 12        | 1                    | buy  | BID              | 12     | 2      |
      | lp1   | ETH/MAR22 | 12        | 1                    | buy  | MID              | 12     | 1      |
      | lp1   | ETH/MAR22 | 12        | 1                    | sell | ASK              | 12     | 2      |
      | lp1   | ETH/MAR22 | 12        | 1                    | sell | MID              | 12     | 1      |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR22"

    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 10   | party2 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 500       | 1500      | 1000         | 1100           | 10            |

    And the liquidity fee factor should be "0.08" for the market "ETH/MAR22"

    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | ETH/MAR22 | lqm-params           | 1e-3                   | 0                         | fees-config-2          |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.01" for the market "ETH/MAR22"

    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | ETH/MAR22 | lqm-params           | 1e-3                   | 0                         | fees-config-3          |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0047272727272727" for the market "ETH/MAR22"

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee   | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 800               | 0.006 | amendment |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0054545454545455" for the market "ETH/MAR22"

    # Use the MARGINAL COST to calculate the liquidity fee. This value with be one of the LP fee values (0042-LIQF-059)
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor | liquidity fee settings |
      | ETH/MAR22 | lqm-params           | 1e-3                   | 0                         | fees-config-4          |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee    | lp type   |
      | lp1 | lp1   | ETH/MAR22 | 800               | 0.0375 | amendment |
    Then the network moves ahead "1" blocks
    And the liquidity fee factor should be "0.0375" for the market "ETH/MAR22"
