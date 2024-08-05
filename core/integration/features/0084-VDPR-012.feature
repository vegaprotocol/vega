Feature: At the start of an epoch, each parties volume_discount_factor is reevaluated and fixed for the epoch (0084-VDPR-012).

  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |

    Given the liquidity monitoring parameters:
      | name       | triggering ratio | time window | scaling factor |
      | lqm-params | 1.0              | 20s         | 1.0            |

    And the following network parameters are set:
      | name                                    | value |
      | market.value.windowLength               | 60s   |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 6     |
      | market.auction.minimumDuration          | 1     |
      | market.fee.factors.infrastructureFee    | 0.001 |
      | market.fee.factors.makerFee             | 0.004 |

    #risk factor short:3.5569036
    #risk factor long:0.801225765
    And the volume discount program tiers named "VDP-01":
      | volume | infra factor | liquidity factor | maker factor |
      | 1000   | 0.001        | 0.002            | 0.003        |
      | 2000   | 0.005        | 0.006            | 0.007        |
      | 3000   | 0.010        | 0.012            | 0.014        |
    And the volume discount program:
      | id  | tiers  | closing timestamp | window length |
      | id1 | VDP-01 | 0                 | 4             |

    And the following assets are registered:
      | id  | decimal places |
      | ETH | 0              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.0004    | 0.001              |
    And the price monitoring named "price-monitoring":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 3                 |

    And the liquidity sla params named "SLA-22":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 0.5         | 0.6                          | 1                             | 1.0                    |

    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model            | margin calculator   | auction duration | fees          | price monitoring | data source config     | linear slippage factor | quadratic slippage factor | sla params |
      | ETH/MAR24 | ETH        | ETH   | lqm-params           | log-normal-risk-model | margin-calculator-1 | 2                | fees-config-1 | price-monitoring | default-eth-for-future | 1e0                    | 0                         | SLA-22     |

    And the following network parameters are set:
      | name                                                | value |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | validators.epoch.length                             | 5s    |
      | market.liquidity.stakeToCcyVolume                   | 1     |
      | market.liquidity.successorLaunchWindowLength        | 1h    |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.7   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.6   |
      | validators.epoch.length                             | 10s   |
      | market.liquidity.earlyExitPenalty                   | 0.25  |

    Given the average block duration is "1"
  @Now
  Scenario: 001: Check that the volume discount factor is updated after each epoch
    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount   |
      | lp1    | ETH   | 10000000 |
      | party1 | ETH   | 10000000 |
      | party2 | ETH   | 10000000 |
      | party3 | ETH   | 10000000 |

    And the parties submit the following liquidity provision:
      | id   | party | market id | commitment amount | fee  | lp type    |
      | lp_1 | lp1   | ETH/MAR24 | 100000            | 0.02 | submission |

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR24 | buy  | 10     | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party1 | ETH/MAR24 | buy  | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/MAR24 | buy  | 100    | 990   | 0                | TYPE_LIMIT | TIF_GTC |
      | lp1    | ETH/MAR24 | sell | 100    | 1010  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR24 | sell | 10     | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR24 | sell | 1      | 1000  | 0                | TYPE_LIMIT | TIF_GTC |

    Then the opening auction period ends for market "ETH/MAR24"
    And the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 1000  | 1    | party2 |
    And the market data for the market "ETH/MAR24" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 3600    | 973       | 1027      | 3556         | 100000         | 1             |
    And the party "party3" has the following discount infra factor "0"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party3 | ETH/MAR24 | buy  | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3 | ETH/MAR24 | sell | 1      | 0     | 1                | TYPE_MARKET | TIF_IOC |
    When the network moves ahead "1" epochs
    And the party "party3" has the following discount infra factor "0.005"
    And the party "party3" has the following discount liquidity factor "0.006"
    And the party "party3" has the following discount maker factor "0.007"

    Then the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type        | tif     |
      | party3 | ETH/MAR24 | buy  | 20     | 0     | 1                | TYPE_MARKET | TIF_IOC |
      | party3 | ETH/MAR24 | sell | 20     | 0     | 1                | TYPE_MARKET | TIF_IOC |
    When the network moves ahead "1" epochs
    And the party "party3" has the following discount infra factor "0.01"
    And the party "party3" has the following discount liquidity factor "0.012"
    And the party "party3" has the following discount maker factor "0.014"
