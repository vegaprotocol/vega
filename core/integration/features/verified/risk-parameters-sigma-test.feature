Feature: test risk model parameter sigma
  Background:

    Given the log normal risk model named "log-normal-risk-model-53":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 50    |
    #risk factor short:999999.0000000
    #risk factor long:1
    And the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 43200   | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees          | price monitoring   | data source config     |
      | ETH/MAR53 | ETH        | USD   | log-normal-risk-model-53 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount                      |
      | party0 | USD   | 500000000000000000000000000 |
      | party1 | USD   | 5000000000                  |
      | party2 | USD   | 5000000000                  |
      | party3 | USD   | 5000000000                  |

  Scenario: 001, test market ETH/MAR53(sigma=50),
    And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR53 | 100000000         | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR53 | 100000000         | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR53 | buy  | 10     | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR53 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR53 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR53 | sell | 10     | 11    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |

    When the opening auction period ends for market "ETH/MAR53"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR53"
    # And the network moves ahead "1" blocks

    And the market data for the market "ETH/MAR53" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_CONTINUOUS | 43200   | 1         | 211       | 9999990      | 100000000      | 1             |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1 x 10 x 1 x 999999.00000000000 =557872

    Then the order book should have the following volumes for market "ETH/MAR53":
      | side | price | volume    |
      | sell | 31    | 22580646  |
      | sell | 11    | 10        |
      | buy  | 9     | 10        |
      | buy  | 1     | 200000001 |

    And the parties should have the following account balances:
      | party  | asset | market id | margin          | general                     | bond      |
      | party0 | USD   | ETH/MAR53 | 270967481032248 | 499999999999729032418967752 | 100000000 |
      | party1 | USD   | ETH/MAR53 | 133             | 4999999867                  | 0         |
      | party2 | USD   | ETH/MAR53 | 131999869       | 4868000131                  | 0         |

    # mentainance margin level for LP: 31*22580646*999999=6.999e14
    # initial  margin level for LP: 31*22580646*999999 *1.5=1.05e15

    And the parties should have the following margin levels:
      | party  | market id | maintenance     | search          | initial         | release         |
      | party0 | ETH/MAR53 | 225806234193540 | 248386857612894 | 270967481032248 | 316128727870956 |