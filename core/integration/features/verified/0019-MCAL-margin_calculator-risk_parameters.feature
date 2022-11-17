Feature: test risk model parameter change in margin calculation
  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 1.0   |
    #risk factor short = 0.16882368861315200
    #risk factor long = 0.145263949

    Given the log normal risk model named "log-normal-risk-model-2":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.001 | 0  | 0 | 2.0   |
    #risk factor short = 0.36483236867768200
    #risk factor long = 0.270133394

    Given the log normal risk model named "log-normal-risk-model-3":
      | risk aversion | tau   | mu | r | sigma |
      | 0.000001      | 0.002 | 0  | 0 | 1.0   |
    #risk factor short = 0.24649034405344100
    #risk factor long = 0.199294303

    Given the log normal risk model named "log-normal-risk-model-4":
      | risk aversion | tau   | mu | r | sigma |
      | 0.0001        | 0.001 | 0  | 0 | 1.0   |
    #risk factor short = 0.13281340025639400
    #risk factor long = 0.118078679

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1000    | 0.99        | 300               |

    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config          |
      | ETH/MAR21 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-2 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
      | ETH/MAR23 | ETH        | USD   | log-normal-risk-model-3 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |
      | ETH/MAR24 | ETH        | USD   | log-normal-risk-model-4 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |

  Scenario: lognormal risk model in 4 markets with different risk parameters , 0010-MARG-012, 0010-MARG-013, 0010-MARG-014

    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR21 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR21 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp2 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp2 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp3 | party0 | ETH/MAR23 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp3 | party0 | ETH/MAR23 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp4 | party0 | ETH/MAR24 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp4 | party0 | ETH/MAR24 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-21  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-22  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-23 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-24 |
      | party1 | ETH/MAR23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-31  |
      | party1 | ETH/MAR23 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-32  |
      | party2 | ETH/MAR23 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-33 |
      | party2 | ETH/MAR23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-34 |
      | party1 | ETH/MAR24 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-41  |
      | party1 | ETH/MAR24 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-42  |
      | party2 | ETH/MAR24 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-43 |
      | party2 | ETH/MAR24 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-44 |

    When the opening auction period ends for market "ETH/MAR21"
    When the opening auction period ends for market "ETH/MAR22"
    When the opening auction period ends for market "ETH/MAR23"
    When the opening auction period ends for market "ETH/MAR24"

    And the market data for the market "ETH/MAR21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 1688         | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.16882368861315200 =1689
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 972       | 1029      | 3648         | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.36483236867768200 = 3648
    And the market data for the market "ETH/MAR24" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 1328         | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.13281340025639400 = 1328
    And the market data for the market "ETH/MAR24" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 1328         | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.13281340025639400 = 1328

    #check the volume on the order book, liquidity price has been kept within price monitoring bounds

    Then the order book should have the following volumes for market "ETH/MAR21":
      | side | price | volume |
      | sell | 1100  | 92     |
      | sell | 1120  | 0      |
      | buy  | 900   | 113    |
      | buy  | 880   | 0      |
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 92     |
      | sell | 1120  | 0      |
      | buy  | 900   | 113    |
      | buy  | 880   | 0      |
    Then the order book should have the following volumes for market "ETH/MAR23":
      | side | price | volume |
      | sell | 1100  | 92     |
      | sell | 1120  | 0      |
      | buy  | 900   | 113    |
      | buy  | 880   | 0      |
    Then the order book should have the following volumes for market "ETH/MAR24":
      | side | price | volume |
      | sell | 1100  | 92     |
      | sell | 1120  | 0      |
      | buy  | 900   | 113    |
      | buy  | 880   | 0      |

    # risk model 001: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR21 | 19524  | 197849   | 50000 |
      | party1 | USD   | ETH/MAR21 | 3117   | 99985185 | 0     |
      | party2 | USD   | ETH/MAR21 | 3429   | 99983148 | 0     |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-900)+10*0.145263949*1000 + 1*0.145263949*1000=2598
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR21 | 16270       | 17897  | 19524   | 22778   |
      | party1 | ETH/MAR21 | 2598        | 2857   | 3117    | 3637    |
      | party2 | ETH/MAR21 | 2858        | 3143   | 3429    | 4001    |

    # risk model 002: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 39840  | 197849   | 50000 |
      | party1 | USD   | ETH/MAR22 | 4766   | 99985185 | 0     |
      | party2 | USD   | ETH/MAR22 | 6016   | 99983148 | 0     |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-900)+10*0.270133394*1000 + 1*0.270133394*1000=3972
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 33200       | 36520  | 39840   | 46480   |
      | party1 | ETH/MAR22 | 3972        | 4369   | 4766    | 5560    |
      | party2 | ETH/MAR22 | 5014        | 5515   | 6016    | 7019    |

    #risk model 003: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR23 | 26917  | 197849   | 50000 |
      | party1 | USD   | ETH/MAR23 | 3831   | 99985185 | 0     |
      | party2 | USD   | ETH/MAR23 | 4454   | 99983148 | 0     |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-900)+10*0.24649034405344100*1000 + 1*0.24649034405344100*1000=3712
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR23 | 22431       | 24674  | 26917   | 31403   |
      | party1 | ETH/MAR23 | 3193        | 3512   | 3831    | 4470    |
      | party2 | ETH/MAR23 | 3712        | 4083   | 4454    | 5196    |

    # risk model 004: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR24 | 15870  | 197849   | 50000 |
      | party1 | USD   | ETH/MAR24 | 3101   | 99985185 | 0     |
      | party2 | USD   | ETH/MAR24 | 2953   | 99983148 | 0     |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-900)+10*0.118078679*1000 + 1*0.118078679*1000=2299
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR24 | 13225       | 14547  | 15870   | 18515   |
      | party1 | ETH/MAR24 | 2299        | 2528   | 2758    | 3218    |
      | party2 | ETH/MAR24 | 2461        | 2707   | 2953    | 3445    |




