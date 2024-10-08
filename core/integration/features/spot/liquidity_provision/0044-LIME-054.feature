Feature: Spot market SLA

  Scenario: 001 0044-LIME-054,For a market that is in opening auction and LP has committed liquidity:
    #- When a LP increases their commitment then:
    # - It takes effect immediately for the purposes of LP stake supplied to the market
    # - In terms of the liquidity they are expected to supply: this only takes effect from the start of the next epoch
    Given time is updated to "2023-07-20T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.999       | 300               |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.6                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 1              |
      | BTC | 1              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 2s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0.4   |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0.4   |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.4   |
      | validators.epoch.length                             | 4s    |

    And the spot markets:
      | id      | name    | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | sla params |
      | BTC/ETH | BTC/ETH | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | SLA-1      |
    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 10000  |
      | party2 | BTC   | 500    |
      | lp1    | ETH   | 4000   |
      | lp1    | BTC   | 60     |
      | lp2    | ETH   | 4000   |
      | lp2    | BTC   | 60     |
      | lp3    | ETH   | 4000   |
      | lp3    | BTC   | 60     |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.8            |

    When the spot markets are updated:
      | id      | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH   | 1000              | 0.1 | submission |
      | lp2 | lp2   | BTC/ETH   | 2000              | 0.1 | submission |
      | lp3 | lp3   | BTC/ETH   | 2000              | 0.1 | submission |

    Then the network moves ahead "1" blocks
    And the network treasury balance should be "0" for the asset "ETH"
    Then the party "lp1" lp liquidity bond account balance should be "1000" for the market "BTC/ETH"
    Then the party "lp2" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"
    And the supplied stake should be "5000" for the market "BTC/ETH"

    #0044-LIME-055: LP can decrease or cancel their commitment and it will take effect immediately without incurring penalties
    When party "lp3" cancels their liquidity provision for market "BTC/ETH"

    Then the network moves ahead "1" blocks

    And the network treasury balance should be "0" for the asset "ETH"
    Then the party "lp1" lp liquidity bond account balance should be "1000" for the market "BTC/ETH"
    Then the party "lp2" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"
    Then the party "lp3" lp liquidity bond account balance should be "0" for the market "BTC/ETH"
    And the supplied stake should be "3000" for the market "BTC/ETH"

    # place orders and generate trades
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH   | buy  | 6      | 8     | 0                | TYPE_LIMIT | TIF_GTC | party-order5 |      |
      | party1 | BTC/ETH   | buy  | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH   | sell | 1      | 15    | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH   | sell | 6      | 24    | 0                | TYPE_LIMIT | TIF_GTC | party-order6 |      |

    When the network moves ahead "1" blocks

    Then the market data for the market "BTC/ETH" should be:
      | mark price | trading mode            | auction trigger             | target stake | supplied stake | open interest |
      | 15         | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 4000         | 3000           | 0             |

    Then the network moves ahead "5" blocks
    And the network treasury balance should be "1200" for the asset "ETH"
    Then the party "lp1" lp liquidity bond account balance should be "600" for the market "BTC/ETH"
    Then the party "lp2" lp liquidity bond account balance should be "1200" for the market "BTC/ETH"
    And the supplied stake should be "1800" for the market "BTC/ETH"

    #0044-LIME-064: When the LP increases its commitment and the increment is higher than its general account balance, the increments are rejected, and the old provision persists.
    When the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | lp type    | error                                            |
      | lp1 | lp1   | BTC/ETH   | 50000             | 0.1 | amendments | commitment submission rejected, not enough stake |
