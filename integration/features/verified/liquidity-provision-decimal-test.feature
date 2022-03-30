Feature: testing decimal when with LP commitment


  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

    And the following assets are registered:
      | id  | decimal places |
      | USD | 5              |
   
    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau    | mu | r     | sigma |
      | 0.001         | 0.0001 | 0  | 0     | 1.5   |

      # RiskFactorShort	0.0516933
      # RiskFactorLong	0.04935184

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 3600    | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | decimal places | position decimal places |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |5               |5                        |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount        |
      | lp     | USD   | 10000000000000 |
      | party1 | USD   | 10000000000000 |
      | party2 | USD   | 10000000000000 |
  
  Scenario: 001, same dp setting: market 5/ asset 5/ position 5; 0070-MKTD-003, 0070-MKTD-004, 0070-MKTD-005, 0070-MKTD-006, 0070-MKTD-007

    Given the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset  | lp type    |
      | lp1 | lp     | ETH/MAR22 | 390500000000      | 0.3   | sell | ASK              | 13         | 100000  | submission |
      | lp1 | lp     | ETH/MAR22 | 390500000000      | 0.3   | buy  | BID              | 2          | 100000  | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price      | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 100000 | 10000000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-3  |
      | party1 | ETH/MAR22 | buy  | 500000 | 90000000   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party1 | ETH/MAR22 | buy  | 500000 | 100100000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party2 | ETH/MAR22 | sell | 500000 | 95100000   | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 500000 | 120000000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party2 | ETH/MAR22 | sell | 100000 | 10000000000| 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "500000" at a price of "95100000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1001 x 5 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 97600000   | TRADING_MODE_CONTINUOUS | 3600    | 93642254  | 101698911  | 25224720     | 390500000000   | 500000        |
 
    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price      | volume    |
      | sell | 110000000  | 0         |
      | sell | 102000000  | 0         |
      | sell | 101000000  | 0         |
      | sell | 120000000  | 651400000 |
      | sell | 120100000  | 0         |
      | buy  | 90000000   | 868300000 |
      | buy  | 99000000   | 0         |
      | buy  | 98000000   | 0         |
    
    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin      | general        | bond         |
      | lp     | USD   | ETH/MAR22 | 50159587363 | 9559340412637  | 390500000000 |

    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search      | initial      | release      |
      | lp     | ETH/MAR22 | 41799656136 | 45979621749 | 50159587363  | 58519518590  |

    # #check position (party0 has no position)
    # Then the parties should have the following profit and loss:
    #   | party  | volume | unrealised pnl | realised pnl |
    #   | party1 | 10     | 0              | 0            |
    #   | party2 | -10    | 0              | 0            |

    # When the parties place the following orders:
    #   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
    #   | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    # And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    # # #check the volume on the order book
    # Then the order book should have the following volumes for market "ETH/MAR22":
    #   | side | price | volume |
    #   | sell | 1100  | 1      |
    #   | sell | 1010  | 101    |
    #   | buy  | 1000  | 130    |
    #   | buy  | 990   | 1      |
    #   | buy  | 900   | 1      |
    # When the parties place the following orders:
    #   | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
    #   | party2 | ETH/MAR22 | sell | 50     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    # And the market data for the market "ETH/MAR22" should be:
    #   | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
    #   | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 213414       | 50000          | 60            |
    # # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1010 x 13 x 1 x 0.1
    # # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction

    # And the parties should have the following account balances:
    #   | party  | asset | market id | margin | general  | bond  |
    #   | party0 | USD   | ETH/MAR22 | 426829 | 23251    | 50000 |
    #   | party1 | USD   | ETH/MAR22 | 12190  | 99987810 | 0     |
    #   | party2 | USD   | ETH/MAR22 | 264754 | 99734946 | 0     |
    # #check the margin levels
    # Then the parties should have the following margin levels:
    #   | party  | market id | maintenance | search | initial | release |
    #   | party0 | ETH/MAR22 | 355691      | 391260 | 426829  | 497967  |

    # Then the order book should have the following volumes for market "ETH/MAR22":
    #   | side | price | volume |
    #   | sell | 1100  | 1      |
    #   | sell | 1010  | 101    |
    #   | sell | 1000  | 0      |
    #   | buy  | 1000  | 0      |
    #   | buy  | 990   | 103    |
    #   | buy  | 900   | 1      |

    # And the parties should have the following account balances:
    #   | party  | asset | market id | margin | general  | bond  |
    #   | party0 | USD   | ETH/MAR22 | 426829 | 23251    | 50000 |
    #   | party1 | USD   | ETH/MAR22 | 12190  | 99987810 | 0     |
    #   | party2 | USD   | ETH/MAR22 | 264754 | 99734946 | 0     |
    #   | party3 | USD   | ETH/MAR22 | 28826  | 99971294 | 0     |

    