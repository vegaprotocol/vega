Feature: Check not penalty if time on book amount is high enough

  Background:
    
    Given the following network parameters are set:
      | name                                                  | value |
      | market.liquidity.bondPenaltyParameter                 | 1     |
      | network.markPriceUpdateMaximumFrequency               | 0s    |
      | limits.markets.maxPeggedOrders                        | 2     |
      | validators.epoch.length                               | 5s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.stakeToCcyVolume                   | 1.0   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.19  |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 1     |
    And the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.10             | 24h         | 1              |  
    
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
      | 0.05        | 0.5                          | 1                             | 1.0                    |
    And the spot markets:
      | id      | name    | base asset | quote asset | liquidity monitoring | risk model            | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lqm-params           | simple-risk-model-1   | 2                | fees-config-1 | price-monitoring-1 | SLA      |
     And the following network parameters are set:
      | name                                               | value |
      | market.liquidity.providersFeeCalculationTimeStep   | 5s    |

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount  |
      | party1 | ETH   | 100000000  |
      | party2 | ETH   | 100000000  |
      | party1 | BTC   | 100000000  |
      | party2 | BTC   | 100000000  |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type     |
      | lp1 | party1 | BTC/ETH   | 1000              | 0.001 | submission  |
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party1 | BTC/ETH   | buy  | 30     | 910   | 0                | TYPE_LIMIT | TIF_GTC | party1-order1 |
      | party1 | BTC/ETH   | sell | 20     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party1-order2 |
      | party2 | BTC/ETH   | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party2-order1 |

  Scenario: If a liquidity provider has fraction_of_time_on_book >= market.liquidity.commitmentMinTimeFraction, no penalty will be taken from their bond account (0044-LIME-102)
    When the opening auction period ends for market "BTC/ETH"
    And the auction ends with a traded volume of "10" at a price of "1000"
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 0            | 1000           | 0             | 910                   | 955              | 1000                    |

    And the parties should have the following account balances:
      | party  | asset | market id | general  | bond |
      | party1 | ETH   | BTC/ETH   | 99981700 | 1000 |    

    # Move forward an epoch and make sure the accounts do not change as we have 5/5 blocks covered
    When the network moves ahead "7" blocks
    Then the liquidity provisions should have the following states:
      | id  | party  | market    | commitment amount | status           |
      | lp1 | party1 | BTC/ETH   | 1000              | STATUS_ACTIVE    |
    And the parties should have the following account balances:
      | party  | asset | market id | general  | bond |
      | party1 | ETH   | BTC/ETH   | 99981700 | 1000 |    

    # Move forward 6 blocks and then cancel the commitment covering order
    When the network moves ahead "6" blocks
    # Cancel the order so we are no longer covering our commitment
    Then the parties cancel the following orders:
      | party   | reference      |
      | party1  | party1-order1  |
      | party1  | party1-order2  |

    # Move forward 1 block to end the epoch and make sure we still do not get punished as we covered 4/5 blocks (0.8 coverage > 0.5 required)
    When the network moves ahead "1" blocks
    And the parties should have the following account balances:
      | party  | asset | market id | general   | bond |
      | party1 | ETH   | BTC/ETH   | 100009000 | 1000 |    
    # Now move forward 7 more blocks to complete another epoch and to show we do get punished
    When the network moves ahead "7" blocks
    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | target stake | supplied stake | open interest | best static bid price | static mid price | best static offer price |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000         | 810            | 0             | 0                     | 0                | 0                       |

    And the parties should have the following account balances:
      | party  | asset | market id | general   | bond |
      | party1 | ETH   | BTC/ETH   | 100009000 | 810  |  
