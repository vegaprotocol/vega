Feature: check behavier when mu=-20, and risk factor short is negative

  Scenario: 001, check party2 which has open short order

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu  | r | sigma |
      | 0.000001      | 0.1 | -20 | 0 | 1.0   |
    #risk factor short: -0.3832902
    #risk factor long: 0.820141635
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1000    | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000000 |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    Then the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 985       | 1013      | 48651        | 5000000        | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1013  | 10506  |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 985   | 10153  |
      | buy  | 900   | 1      |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 50     | 0              | 0            |
      | party2 | -50    | 0              | 0            |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party1 | ETH/MAR22 | 53348       | 58682  | 64017   | 74687   |
      | party2 | ETH/MAR22 | 0           | 0      | 0       | 0       |
    #party2 has position and open orders, so the margin account should be non-zero
    Then the parties should have the following account balances:
      | party  | asset | market id | margin | general   |
      | party1 | USD   | ETH/MAR22 | 64017  | 99935983  |
      | party2 | USD   | ETH/MAR22 | 0      | 100000000 |


