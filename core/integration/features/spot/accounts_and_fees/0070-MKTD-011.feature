Feature: Spot market SLA

  Scenario: 001 0070-MKTD-010
    Given time is updated to "2023-07-20T00:00:00Z"

    Given the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    Given the log normal risk model named "lognormal-risk-model-1":
      | risk aversion | tau  | mu | r   | sigma |
      | 0.001         | 0.01 | 0  | 0.0 | 1.2   |

    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.999       | 300               |

    And the liquidity sla params named "SLA-1":
      | price range | commitment min time fraction | performance hysteresis epochs | sla competition factor |
      | 1           | 0.6                          | 2                             | 0.2                    |

    Given the following assets are registered:
      | id  | decimal places |
      | ETH | 2              |
      | BTC | 2              |

    And the following network parameters are set:
      | name                                                | value |
      | network.markPriceUpdateMaximumFrequency             | 2s    |
      | market.liquidity.earlyExitPenalty                   | 0.25  |
      | market.liquidity.bondPenaltyParameter               | 0.2   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0     |
      | market.liquidity.sla.nonPerformanceBondPenaltyMax   | 0     |
      | market.liquidity.maximumLiquidityFeeFactorLevel     | 0.4   |
      | validators.epoch.length                             | 2s    |

    And the spot markets:
      | id          | name      | base asset | quote asset | risk model             | auction duration | fees          | price monitoring   | decimal places | position decimal places | sla params |
      | BTC/ETH_D11 | BTC/ETH11 | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 1              | 1                       | SLA-1      |
      | BTC/ETH_D10 | BTC/ETH11 | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 1              | 0                       | SLA-1      |
      | BTC/ETH_D21 | BTC/ETH11 | BTC        | ETH         | lognormal-risk-model-1 | 1                | fees-config-1 | price-monitoring-1 | 2              | 1                       | SLA-1      |

    And the following network parameters are set:
      | name                                             | value |
      | market.liquidity.providersFeeCalculationTimeStep | 1s    |
      | market.liquidity.stakeToCcyVolume                | 1     |

    Given the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party1 | ETH   | 300000 |
      | party2 | BTC   | 15000  |
      | lp1    | ETH   | 120000 |
      | lp1    | BTC   | 1800   |
      | lp2    | ETH   | 120000 |
      | lp2    | BTC   | 1800   |

    And the average block duration is "1"

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.2              | 20s         | 0.8            |

    When the spot markets are updated:
      | id          | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | BTC/ETH_D11 | updated-lqm-params   | 0.5                    | 0.5                       |
      | BTC/ETH_D10 | updated-lqm-params   | 0.5                    | 0.5                       |
      | BTC/ETH_D21 | updated-lqm-params   | 0.5                    | 0.5                       |

    When the parties submit the following liquidity provision:
      | id  | party | market id   | commitment amount | fee | lp type    |
      | lp1 | lp1   | BTC/ETH_D11 | 3000              | 0.1 | submission |
      | lp2 | lp2   | BTC/ETH_D11 | 3000              | 0.1 | submission |
      | lp3 | lp1   | BTC/ETH_D10 | 3000              | 0.1 | submission |
      | lp4 | lp2   | BTC/ETH_D10 | 3000              | 0.1 | submission |
      | lp5 | lp1   | BTC/ETH_D21 | 3000              | 0.1 | submission |
      | lp6 | lp2   | BTC/ETH_D21 | 3000              | 0.1 | submission |

    Then the network moves ahead "1" blocks

    Then the market data for the market "BTC/ETH_D11" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 480          | 6000           | 0             |
    Then the market data for the market "BTC/ETH_D10" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 4800         | 6000           | 0             |
    Then the market data for the market "BTC/ETH_D21" should be:
      | mark price | trading mode                 | auction trigger         | target stake | supplied stake | open interest |
      | 0          | TRADING_MODE_OPENING_AUCTION | AUCTION_TRIGGER_OPENING | 480          | 6000           | 0             |

    #0070-MKTD-010:As a user all orders placed (either directly or through LP) are shown in events with prices in market precision
    And the parties place the following orders:
      | party  | market id   | side | volume | price | resulting trades | type       | tif     | reference    | only |
      | party1 | BTC/ETH_D11 | buy  | 60     | 80    | 0                | TYPE_LIMIT | TIF_GTC | party-order5 |      |
      | party1 | BTC/ETH_D11 | buy  | 10     | 150   | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH_D11 | sell | 10     | 150   | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH_D11 | sell | 60     | 240   | 0                | TYPE_LIMIT | TIF_GTC | party-order6 |      |
      | party1 | BTC/ETH_D10 | buy  | 6      | 80    | 0                | TYPE_LIMIT | TIF_GTC | party-order5 |      |
      | party1 | BTC/ETH_D10 | buy  | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH_D10 | sell | 1      | 150   | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH_D10 | sell | 6      | 240   | 0                | TYPE_LIMIT | TIF_GTC | party-order6 |      |
      | party1 | BTC/ETH_D21 | buy  | 60     | 800   | 0                | TYPE_LIMIT | TIF_GTC | party-order5 |      |
      | party1 | BTC/ETH_D21 | buy  | 10     | 1500  | 0                | TYPE_LIMIT | TIF_GTC | party-order3 |      |
      | party2 | BTC/ETH_D21 | sell | 10     | 1500  | 0                | TYPE_LIMIT | TIF_GTC | party-order4 |      |
      | party2 | BTC/ETH_D21 | sell | 60     | 2400  | 0                | TYPE_LIMIT | TIF_GTC | party-order6 |      |

    When the network moves ahead "2" blocks
    #0070-MKTD-015:Trades prices, like orders, are shown in market precision. The transfers and margin requirements are in asset precision.
    Then the following trades should be executed:
      | buyer  | price | size | seller |
      | party1 | 150   | 10   | party2 |
      | party1 | 150   | 1    | party2 |
      | party1 | 1500  | 10   | party2 |

    Then the market data for the market "BTC/ETH_D11" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3600    | 144       | 156       | 480          | 6000           | 0             |
    Then the market data for the market "BTC/ETH_D10" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 150        | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3600    | 144       | 156       | 4800         | 6000           | 0             |
    #0070-MKTD-012: As a user I should see the market data prices using market precision.
    #0070-MKTD-013: Price bounds are calculated in asset precision, but enforced rounded to the closest value in market precision in range
    Then the market data for the market "BTC/ETH_D21" should be:
      | mark price | trading mode            | auction trigger             | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1500       | TRADING_MODE_CONTINUOUS | AUCTION_TRIGGER_UNSPECIFIED | 3600    | 1438      | 1564      | 480          | 6000           | 0             |
    Then "lp1" should have general account balance of "111000" for asset "ETH"
    Then "lp2" should have general account balance of "111000" for asset "ETH"

    When the parties submit the following liquidity provision:
      | id  | party | market id   | commitment amount | fee | lp type   |
      | lp1 | lp1   | BTC/ETH_D11 | 2000              | 0.1 | amendment |
      | lp4 | lp1   | BTC/ETH_D10 | 2000              | 0.1 | amendment |
      | lp5 | lp1   | BTC/ETH_D21 | 2000              | 0.1 | amendment |

    Then the network moves ahead "4" blocks
    When the parties submit the following liquidity provision:
      | id  | party | market id   | commitment amount | fee | lp type   |
      | lp2 | lp2   | BTC/ETH_D11 | 2000              | 0.1 | amendment |
      | lp4 | lp2   | BTC/ETH_D10 | 2000              | 0.1 | amendment |
      | lp6 | lp2   | BTC/ETH_D21 | 2000              | 0.1 | amendment |

    Then the network moves ahead "4" blocks

    #bond penalty for lp2 = 800*0.25 =200
    And the network treasury balance should be "200" for the asset "ETH"
# Then the party "lp1" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"
# Then the party "lp2" lp liquidity bond account balance should be "2000" for the market "BTC/ETH"
# Then "lp1" should have general account balance of "2000" for asset "ETH"
# Then "lp2" should have general account balance of "1800" for asset "ETH"


