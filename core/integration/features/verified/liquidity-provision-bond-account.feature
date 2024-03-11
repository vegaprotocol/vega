Feature: Replicate LP getting distressed during continuous trading, check if penalty is implemented correctly

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | lqm-params         | 0.24             | 24h         | 1.0            |
    And the markets:
      | id        | quote name | asset | liquidity monitoring | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor | sla params      |
      | ETH/MAR22 | ETH        | USD   | lqm-params           | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 0.7                    | 0                         | default-futures |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | party4 | USD   | 100000000 |
      | party5 | USD   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |
      | limits.markets.maxPeggedOrders          | 2     |
      | market.liquidity.bondPenaltyParameter               | 0    |
      | validators.epoch.length                               | 5s   |
      | market.liquidity.sla.nonPerformanceBondPenaltySlope | 0    |
    

  @Now @NoPerp
  Scenario: 001, LP gets distressed during continuous trading, no DPD setting (0044-LIME-002, 0035-LIQM-004)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.24             | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | updated-lqm-params   | 0.7                    | 0                         |
    And the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 0.2   |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount |
      | party0 | USD   | 12500  |
    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | amendment  |
    And the parties place the following pegged iceberg orders:
      | party  | market id | peak size | minimum visible size | side | pegged reference | volume  | offset |
      | party0 | ETH/MAR22 | 49        | 1                    | sell | ASK              | 49      | 20     |
      | party0 | ETH/MAR22 | 52        | 1                    | buy  | BID              | 52      | 20     |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party4 | ETH/MAR22 | buy  | 100    | 850   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party5 | ETH/MAR22 | sell | 100    | 1200  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-5 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
# target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 3.5569
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 50000          | 10            |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 253354   | 50000 |
      | party1 | USD   | ETH/MAR22 | 19930  | 99980070 |       |
      | party2 | USD   | ETH/MAR22 | 59619  | 99940381 |       |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/MAR22 | 174289      |
      | party1 | ETH/MAR22 | 16609       |
      | party2 | ETH/MAR22 | 49683       |
    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 142276       | 50000          | 40            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 40 x 1 x 3.5569=142276

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 253354   | 50000 |
      | party1 | USD   | ETH/MAR22 | 19930  | 99980070 |       |
      | party2 | USD   | ETH/MAR22 | 273034 | 99726786 |       |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/MAR22 | 174289      |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    # update linear slippage factor more in line with what book-based slippage used to be
    And the markets are updated:
      | id          | linear slippage factor |
      | ETH/MAR22   | 0.05                   |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party0 | ETH/MAR22 | sell | 70     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party0-sell-3 |
      | party1 | ETH/MAR22 | buy  | 100    | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy-4  |

    Then the network moves ahead "1" blocks

    Then the parties should have the following margin levels:
      | party  | market id | maintenance |
      | party0 | ETH/MAR22 | 426772      |
      | party1 | ETH/MAR22 | 94682       |

    And the insurance pool balance should be "9085" for the market "ETH/MAR22"

    #check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general | bond |
      | party0 | USD   | ETH/MAR22 | 503415 | 0       | 280  |
