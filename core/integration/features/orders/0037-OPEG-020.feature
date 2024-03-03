Feature: 0037-OPEG-020, 0037-OPEG-021

  Scenario: Given a mid-price which is not an integer multiple of the market tick size, a buy/sell order pegged to the mid price should have it's price rounded up to the nearest market tick size

  Background:
    Given the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 1     |
      | network.markPriceUpdateMaximumFrequency             | 0s    |
      | limits.markets.maxPeggedOrders                      | 6     |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.stakeToCcyVolume                   | 1.0   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |

    And the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 0.1              | 24h         | 1              |

    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |

    And the average block duration is "1"
    And the simple risk model named "simple-risk-model-1":
      | long | short | max move up | min move down | probability of trading |
      | 0.1  | 0.1   | 60          | 50            | 0.2                    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 5                 |
    And the liquidity sla params named "SLA":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.01        | 0.5                          | 1                             | 1.0                    |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model          | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params | decimal places | tick size |
      | ETH/DEC21 | ETH        | ETH   | lqm-params           | simple-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.5                    | 0                         | SLA        | 0              | 10        |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party1 | ETH   | 100000000 |
      | party3 | ETH   | 100000000 |
      | party4 | ETH   | 100000000 |
    And the average block duration is "1"
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party1 | ETH/DEC21 | 10000             | 0.001 | submission |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party3 | ETH/DEC21 | buy  | 100    | 100   | 0                | TYPE_LIMIT | TIF_GTC | p3b1      |
      | party3 | ETH/DEC21 | buy  | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | p3b2      |
      | party4 | ETH/DEC21 | sell | 10     | 110   | 0                | TYPE_LIMIT | TIF_GTC | p4s2      |
      | party4 | ETH/DEC21 | sell | 1000   | 190   | 0                | TYPE_LIMIT | TIF_GTC | p4s1      |

    Then the network moves ahead "2" blocks
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/DEC21"

    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party3 | 110   | 10   | party4 |

    # Place PEGGED orders to cover our commitment
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     | pegged reference | pegged offset |
      | party1 | ETH/DEC21 | buy  | 10     | 0     | 0                | TYPE_LIMIT | TIF_GTC | party1-order1 | MID              | 10            |
      | party1 | ETH/DEC21 | sell | 10     | 0     | 0                | TYPE_LIMIT | TIF_GTC | party1-order2 | MID              | 10            |

    Then the order book should have the following volumes for market "ETH/DEC21":
      | side | price | volume |
      | buy  | 130   | 10     |
      | sell | 160   | 10     |



