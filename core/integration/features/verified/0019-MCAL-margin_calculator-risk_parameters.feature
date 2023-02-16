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
    And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR21 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-2 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR23 | ETH        | USD   | log-normal-risk-model-3 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR24 | ETH        | USD   | log-normal-risk-model-4 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |

  Scenario: lognormal risk model in 4 markets with different risk parameters , 0010-MARG-012, 0010-MARG-013, 0010-MARG-014

    Given the average block duration is "1"

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
      | sell | 1120  | 45     |
      | sell | 1100  | 1      |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1120  | 45     |
      | sell | 1100  | 1      |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR23":
      | side | price | volume |
      | sell | 1120  | 45     |
      | sell | 1100  | 1      |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR24":
      | side | price | volume |
      | sell | 1120  | 45     |
      | sell | 1100  | 1      |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |

    # risk model 001: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR21 | 9937   | 248653   | 50000 |
      | party1 | USD   | ETH/MAR21 | 3333   | 99984664 |       |
      | party2 | USD   | ETH/MAR21 | 3645   | 99982284 |       |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-882)+10*0.145263949*1000 + 1*0.145263949*1000=2778
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR21 | 8281        | 9109   | 9937    | 11593   |
      | party1 | ETH/MAR21 | 2778        | 3055   | 3333    | 3889    |
      | party2 | ETH/MAR21 | 3038        | 3341   | 3645    | 4253    |

    # risk model 002: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 19701  | 248653   | 50000 |
      | party1 | USD   | ETH/MAR22 | 4982   | 99984664 |       |
      | party2 | USD   | ETH/MAR22 | 6232   | 99982284 |       |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-882)+10*0.270133394*1000 + 1*0.270133394*1000=4152
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 16418       | 18059  | 19701   | 22985   |
      | party1 | ETH/MAR22 | 4152        | 4567   | 4982    | 5812    |
      | party2 | ETH/MAR22 | 5194        | 5713   | 6232    | 7271    |

    #risk model 003: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR23 | 13632  | 248653   | 50000 |
      | party1 | USD   | ETH/MAR23 | 4047   | 99984664 |       |
      | party2 | USD   | ETH/MAR23 | 4670   | 99982284 |       |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-882)+10*0.199294303*1000 + 1*0.199294303*1000=3373
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR23 | 11360       | 12496  | 13632   | 15904   |
      | party1 | ETH/MAR23 | 3373        | 3710   | 4047    | 4722    |
      | party2 | ETH/MAR23 | 3892        | 4281   | 4670    | 5448    |
      # | party1 | ETH/MAR23 | 3193        | 3512   | 3831    | 4470    |
      # | party2 | ETH/MAR23 | 3712        | 4083   | 4454    | 5196    |
    # risk model 004: check the required balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR24 | 8077   | 248653   | 50000 |
      | party1 | USD   | ETH/MAR24 | 2974   | 99984664 |       |
      | party2 | USD   | ETH/MAR24 | 3169   | 99982284 |       |

    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-882)+10*0.118078679*1000 + 1*0.118078679*1000=2479
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR24 | 6731        | 7404   | 8077    | 9423    |
      | party1 | ETH/MAR24 | 2479        | 2726   | 2974    | 3470    |
      | party2 | ETH/MAR24 | 2641        | 2905   | 3169    | 3697    |


