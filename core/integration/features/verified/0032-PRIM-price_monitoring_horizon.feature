Feature: 0032-PRIM-price-mornitoring, test horizon trigger.
  Scenario: 001, horizon set to 3600 in price monitoring model.  0032-PRIM-001, 0032-PRIM-009, 0018-RSKM-007

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 5              |

    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau    | mu | r | sigma |
      | 0.001         | 0.0001 | 0  | 0 | 1.5   |

    # RiskFactorShort 0.0516933
    # RiskFactorLong  0.04935184

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 5              | 5                       | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount         |
      | lp     | USD   | 10000000000000 |
      | party1 | USD   | 10000000000000 |
      | party2 | USD   | 10000000000000 |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/MAR22 | 390500000000      | 0.3 | sell | ASK              | 13         | 100000 | submission |
      | lp1 | lp    | ETH/MAR22 | 390500000000      | 0.3 | buy  | BID              | 2          | 100000 | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price       | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 100000 | 10000000    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party1 | ETH/MAR22 | buy  | 500000 | 90000000    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/MAR22 | buy  | 500000 | 100100000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | sell | 500000 | 95100000    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 500000 | 120000000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/MAR22 | sell | 100000 | 10000000000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "500000" at a price of "97600000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1001 x 5 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 97600000   | TRADING_MODE_CONTINUOUS | 3600    | 93642254  | 101698911 | 25224720     | 390500000000   | 500000        |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price       | volume    |
      | buy  | 10000000    | 100000    |
      | buy  | 90000000    | 500000    |
      | buy  | 89900000    | 434371524 |
      | sell | 120000000   | 500000    |
      | sell | 120100000   | 325145712 |
      | sell | 10000000000 | 100000    |

    # check the requried balances
    And the parties should have the following account balances:
      | party | asset | market id | margin      | general       | bond         |
      | lp    | USD   | ETH/MAR22 | 25107048175 | 9584392951825 | 390500000000 |

    #check the margin levels
    Then the parties should have the following margin levels:
      | party | market id | maintenance | search      | initial     | release     |
      | lp    | ETH/MAR22 | 20922540146 | 23014794160 | 25107048175 | 29291556204 |

  Scenario: 002, horizon set to 360000 in price monitoring model.  0032-PRIM-001, 0032-PRIM-009

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 5              |

    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau    | mu | r | sigma |
      | 0.001         | 0.0001 | 0  | 0 | 1.5   |

    # RiskFactorShort 0.0516933
    # RiskFactorLong  0.04935184

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 5              | 5                       | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount         |
      | lp     | USD   | 10000000000000 |
      | party1 | USD   | 10000000000000 |
      | party2 | USD   | 10000000000000 |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/MAR22 | 390500000000      | 0.3 | sell | ASK              | 13         | 100000 | submission |
      | lp1 | lp    | ETH/MAR22 | 390500000000      | 0.3 | buy  | BID              | 2          | 100000 | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price       | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 100000 | 10000000    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party1 | ETH/MAR22 | buy  | 500000 | 90000000    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/MAR22 | buy  | 500000 | 100100000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | sell | 500000 | 95100000    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 500000 | 120000000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/MAR22 | sell | 100000 | 10000000000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "500000" at a price of "97600000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1001 x 5 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 97600000   | TRADING_MODE_CONTINUOUS | 360000  | 63775516  | 145578912 | 25224720     | 390500000000   | 500000        |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price       | volume    |
      | buy  | 10000000    | 100000    |
      | buy  | 90000000    | 500000    |
      | buy  | 89900000    | 434371524 |
      | sell | 120000000   | 500000    |
      | sell | 120100000   | 325145712 |
      | sell | 10000000000 | 100000    |

    # check the requried balances
    And the parties should have the following account balances:
      | party | asset | market id | margin      | general       | bond         |
      | lp    | USD   | ETH/MAR22 | 25107048175 | 9584392951825 | 390500000000 |

    #check the margin levels
    Then the parties should have the following margin levels:
      | party | market id | maintenance | search      | initial     | release     |
      | lp    | ETH/MAR22 | 20922540146 | 23014794160 | 25107048175 | 29291556204 |

  Scenario: 003, horizon set to 360000 in price monitoring model.  0032-PRIM-001, 0032-PRIM-009; 0070-MKTD-003; 0070-MKTD-004; 0070-MKTD-005; 0070-MKTD-006; 0070-MKTD-007; 0070-MKTD-008
    # test different dps, asset: 6/market: 2/position: 1

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 6              |

    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau    | mu | r | sigma |
      | 0.001         | 0.0001 | 0  | 0 | 1.5   |

    # RiskFactorShort 0.0516933
    # RiskFactorLong  0.04935184

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 360000  | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2              | 1                       | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount          |
      | lp     | USD   | 100000000000000 |
      | party1 | USD   | 100000000000000 |
      | party2 | USD   | 100000000000000 |

    Given the parties submit the following liquidity provision:
      | id  | party | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | lp    | ETH/MAR22 | 390500000000      | 0.3 | sell | ASK              | 13         | 10     | submission |
      | lp1 | lp    | ETH/MAR22 | 390500000000      | 0.3 | buy  | BID              | 2          | 10     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price   | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 1000    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party1 | ETH/MAR22 | buy  | 5      | 9000    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/MAR22 | buy  | 5      | 10010   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | sell | 5      | 9510    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 5      | 12000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/MAR22 | sell | 1      | 1000000 | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "5" at a price of "9760"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1001 x 5 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 9760       | TRADING_MODE_CONTINUOUS | 360000  | 6378      | 14557     | 2522472      | 390500000000   | 5             |

    #check the volume on the order book

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price   | volume |
      | buy  | 1000    | 1      |
      | buy  | 9000    | 5      |
      | buy  | 8990    | 43438  |
      | sell | 12000   | 5      |
      | sell | 12010   | 32515  |
      | sell | 1000000 | 1      |

    # check the requried balances
    And the parties should have the following account balances:
      | party | asset | market id | margin      | general        | bond         |
      | lp    | USD   | ETH/MAR22 | 25107538095 | 99584392461905 | 390500000000 |

    #check the margin levels
    Then the parties should have the following margin levels:
      | party | market id | maintenance | search      | initial     | release     |
      | lp    | ETH/MAR22 | 20922948413 | 23015243254 | 25107538095 | 29292127778 |

