Feature: test risk model parameter ranges
  Background:

    Given the margin calculator named "margin-calculator-1":
      | search factor | initial factor | release factor |
      | 1.2           | 1.5            | 1.7            |
    Given the log normal risk model named "log-normal-risk-model-0":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765

    # test risk aversion
    Given the log normal risk model named "log-normal-risk-model-11":
      | risk aversion | tau | mu | r | sigma |
      | 0.00000001    | 0.1 | 0  | 0 | 1.0   |
    #risk factor short:4.9256840
    #risk factor long:0.847272675
    Given the log normal risk model named "log-normal-risk-model-12":
      | risk aversion | tau | mu | r | sigma |
      | 0.1           | 0.1 | 0  | 0 | 1.0   |
    #risk factor short:0.67191327433
    #risk factor long:0.44953951995

    # test tau:1e-8<=x<=1
    Given the log normal risk model named "log-normal-risk-model-21":
      | risk aversion | tau        | mu | r | sigma |
      | 0.000001      | 0.00000001 | 0  | 0 | 1.0   |
    #risk factor short:0.0004950
    #risk factor long:0.000494716
    Given the log normal risk model named "log-normal-risk-model-22":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    Given the log normal risk model named "log-normal-risk-model-23":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 1   | 0  | 0 | 1.0   |
    #risk factor short:86.2176101 (it can not end the auction so this is tested separately in scenario 002)
    #risk factor long:0.996594553

    # test mu:-1e-6<=x<=1e-6
    Given the log normal risk model named "log-normal-risk-model-31":
      | risk aversion | tau | mu        | r | sigma |
      | 0.000001      | 0.1 | -0.000001 | 0 | 1.0   |
    #risk factor short:3.55690313589
    #risk factor long:0.80072822791
    # actual mu = -0.0000001
    Given the log normal risk model named "log-normal-risk-model-32":
      | risk aversion | tau | mu       | r | sigma |
      | 0.000001      | 0.1 | 0.000001 | 0 | 1.0   |
    #risk factor short:3.55690313589
    #risk factor long:0.80072822791
    # actual mu = 0.0000001

    # test r:-1<=x<=1
    Given the log normal risk model named "log-normal-risk-model-41":
      | risk aversion | tau | mu | r  | sigma |
      | 0.000001      | 0.1 | 0  | -1 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    Given the log normal risk model named "log-normal-risk-model-42":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 0.5 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765
    Given the log normal risk model named "log-normal-risk-model-43":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 1 | 1.0   |
    #risk factor short:3.5569036
    #risk factor long:0.801225765

    # test sigma: 1e-3<=x<=100
    Given the log normal risk model named "log-normal-risk-model-51":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 0.001 |
    #risk factor short:0.00156597682
    #risk factor long:0.00156362469

    Given the log normal risk model named "log-normal-risk-model-52":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 10    |
    #risk factor short:55787.28815617000
    #risk factor long:0.99999999877
    # tested in scenario 003
    Given the log normal risk model named "log-normal-risk-model-53":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 100   |
    #risk factor short:999999.00000000000
    #risk factor long:1.00000000000
    # tested in scenario 004

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 43200   | 0.99        | 300               |

    And the markets:
      | id        | quote name | asset | risk model               | margin calculator         | auction duration | fees          | price monitoring   | data source config     | linear slippage factor | quadratic slippage factor |
      | ETH/MAR0  | ETH        | USD   | log-normal-risk-model-0  | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR11 | ETH        | USD   | log-normal-risk-model-11 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR12 | ETH        | USD   | log-normal-risk-model-12 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR21 | ETH        | USD   | log-normal-risk-model-21 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-22 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR23 | ETH        | USD   | log-normal-risk-model-23 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR31 | ETH        | USD   | log-normal-risk-model-31 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR32 | ETH        | USD   | log-normal-risk-model-32 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR41 | ETH        | USD   | log-normal-risk-model-41 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR42 | ETH        | USD   | log-normal-risk-model-42 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR43 | ETH        | USD   | log-normal-risk-model-43 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR51 | ETH        | USD   | log-normal-risk-model-51 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR52 | ETH        | USD   | log-normal-risk-model-52 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
      | ETH/MAR53 | ETH        | USD   | log-normal-risk-model-53 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount         |
      | party0 | USD   | 50000000000000 |
      | party1 | USD   | 50000000000000 |
      | party2 | USD   | 50000000000000 |
      | party3 | USD   | 50000000000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @Now
  Scenario: 001, test different value of risk parameters within defined ranges in different market, AC: 0018-RSKM-001

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.1              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/MAR0  | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR11 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR12 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR21 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR22 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR23 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR31 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR32 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR41 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR42 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR43 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR51 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR52 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR53 | updated-lqm-params   | 1e6                    | 1e6                       |

    And the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 0.2   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id   | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1  | party0 | ETH/MAR0  | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1  | party0 | ETH/MAR0  | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp2  | party0 | ETH/MAR11 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp2  | party0 | ETH/MAR11 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp3  | party0 | ETH/MAR12 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp3  | party0 | ETH/MAR12 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp4  | party0 | ETH/MAR21 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp4  | party0 | ETH/MAR21 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp5  | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp5  | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |
      | lp6  | party0 | ETH/MAR31 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp6  | party0 | ETH/MAR31 | 50000             | 0.001 | sell | ASK              | 500        | 20     | amendment  |
      | lp7  | party0 | ETH/MAR32 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp7  | party0 | ETH/MAR32 | 50000             | 0.001 | sell | ASK              | 500        | 20     | amendment  |
      | lp9  | party0 | ETH/MAR41 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp9  | party0 | ETH/MAR41 | 50000             | 0.001 | sell | ASK              | 500        | 20     | amendment  |
      | lp10 | party0 | ETH/MAR42 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp10 | party0 | ETH/MAR42 | 50000             | 0.001 | sell | ASK              | 500        | 20     | amendment  |
      | lp11 | party0 | ETH/MAR43 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp11 | party0 | ETH/MAR43 | 50000             | 0.001 | sell | ASK              | 500        | 20     | amendment  |
      | lp12 | party0 | ETH/MAR51 | 50000             | 0.001 | buy  | BID              | 500        | 20     | submission |
      | lp12 | party0 | ETH/MAR51 | 50000             | 0.001 | sell | ASK              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR0  | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR0  | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR0  | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR0  | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |
      | party1 | ETH/MAR11 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-21  |
      | party1 | ETH/MAR11 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-22  |
      | party2 | ETH/MAR11 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-23 |
      | party2 | ETH/MAR11 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-24 |
      | party1 | ETH/MAR12 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-31  |
      | party1 | ETH/MAR12 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-32  |
      | party2 | ETH/MAR12 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-33 |
      | party2 | ETH/MAR12 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-34 |
      | party1 | ETH/MAR21 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-41  |
      | party1 | ETH/MAR21 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-42  |
      | party2 | ETH/MAR21 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-43 |
      | party2 | ETH/MAR21 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-44 |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |
      | party1 | ETH/MAR31 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-21  |
      | party1 | ETH/MAR31 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-22  |
      | party2 | ETH/MAR31 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-23 |
      | party2 | ETH/MAR31 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-24 |
      | party1 | ETH/MAR32 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-31  |
      | party1 | ETH/MAR32 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-32  |
      | party2 | ETH/MAR32 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-33 |
      | party2 | ETH/MAR32 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-34 |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR41 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR41 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR41 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR41 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |
      | party1 | ETH/MAR42 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-21  |
      | party1 | ETH/MAR42 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-22  |
      | party2 | ETH/MAR42 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-23 |
      | party2 | ETH/MAR42 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-24 |
      | party1 | ETH/MAR43 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-31  |
      | party1 | ETH/MAR43 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-32  |
      | party2 | ETH/MAR43 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-33 |
      | party2 | ETH/MAR43 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-34 |
      | party1 | ETH/MAR51 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-41  |
      | party1 | ETH/MAR51 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-42  |
      | party2 | ETH/MAR51 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-43 |
      | party2 | ETH/MAR51 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-44 |

    When the opening auction period ends for market "ETH/MAR0"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR0"
    When the opening auction period ends for market "ETH/MAR11"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR11"
    When the opening auction period ends for market "ETH/MAR12"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR12"
    When the opening auction period ends for market "ETH/MAR21"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR21"
    When the opening auction period ends for market "ETH/MAR22"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"
    When the opening auction period ends for market "ETH/MAR31"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR31"
    When the opening auction period ends for market "ETH/MAR32"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR32"

    When the opening auction period ends for market "ETH/MAR41"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR41"
    When the opening auction period ends for market "ETH/MAR42"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR42"
    When the opening auction period ends for market "ETH/MAR43"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR43"

    When the opening auction period ends for market "ETH/MAR51"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR51"

    And the market data for the market "ETH/MAR0" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 3.5569036 =35569
    And the market data for the market "ETH/MAR11" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 49256        | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 4.9256840 = 49256
    And the market data for the market "ETH/MAR12" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 6719         | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.67191327433 = 6719
    And the market data for the market "ETH/MAR21" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 4            | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.0004950 = 4
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 3.5569036 =35569
    And the market data for the market "ETH/MAR31" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 3.55690313589 = 35569
    And the market data for the market "ETH/MAR32" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 3.55690313589 = 35569

    And the market data for the market "ETH/MAR41" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.16882368861315200 =1689
    And the market data for the market "ETH/MAR42" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.36483236867768200 = 3648
    And the market data for the market "ETH/MAR43" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 35569        | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.13281340025639400 = 1328
    And the market data for the market "ETH/MAR51" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 1000      | 1000      | 15           | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.00156597682 = 15

    Then the order book should have the following volumes for market "ETH/MAR0":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR11":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR12":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR21":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR31":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR32":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR41":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR42":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |
    Then the order book should have the following volumes for market "ETH/MAR43":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |

    Then the order book should have the following volumes for market "ETH/MAR51":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 45     |
      | buy  | 900   | 1      |
      | buy  | 880   | 57     |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR0  | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR0  | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR0  | 47378  | 49999999589597 |       |
    # intial margin level for LP = 92*1000*1.2*3.5569036=392682

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR11 | 265987 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR11 | 12595  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR11 | 65611  | 49999999589597 |       |
    # intial margin level for LP = 92*1000*1.2*4.9256840 =543796

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR12 | 36284  | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR12 | 7350   | 49999999893293 |       |
      | party2 | USD   | ETH/MAR12 | 10286  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR21 | 34     | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR21 | 1423   | 49999999893293 |       |
      | party2 | USD   | ETH/MAR21 | 1423   | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR22 | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR22 | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR22 | 47378  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR31 | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR31 | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR31 | 47378  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR32 | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR32 | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR32 | 47378  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR41 | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR41 | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR41 | 47378  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR42 | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR42 | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR42 | 47378  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR43 | 192073 | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR43 | 11986  | 49999999893293 |       |
      | party2 | USD   | ETH/MAR43 | 47378  | 49999999589597 |       |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general        | bond  |
      | party0 | USD   | ETH/MAR51 | 108    | 49999997803076 | 50000 |
      | party1 | USD   | ETH/MAR51 | 1437   | 49999999893293 |       |
      | party2 | USD   | ETH/MAR51 | 1437   | 49999999589597 |       |

  @Now
  Scenario: 002, test market ETH/MAR23 (tau=1)

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.1              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/MAR0  | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR11 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR12 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR21 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR22 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR23 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR32 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR31 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR41 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR42 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR43 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR51 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR52 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR53 | updated-lqm-params   | 1e6                    | 1e6                       |

    And the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 0.2   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR23 | 5000000           | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR23 | 5000000           | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR23 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR23 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR23 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR23 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |

    When the opening auction period ends for market "ETH/MAR23"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR23"

    And the market data for the market "ETH/MAR23" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 43200   | 909       | 1099      | 862176       | 5000000        | 10            |
    #target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 86.2176101 =862176

    Then the order book should have the following volumes for market "ETH/MAR23":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1120  | 4465   |
      | buy  | 900   | 1      |
      | buy  | 880   | 5682   |
    And the parties should have the following account balances:
      | party  | asset | market id | margin    | general        | bond    |
      | party0 | USD   | ETH/MAR23 | 461953956 | 49999533046044 | 5000000 |
      | party1 | USD   | ETH/MAR23 | 14558     | 49999999985442 |         |
      | party2 | USD   | ETH/MAR23 | 1148419   | 49999998851581 |         |

  # initial margin level for LP = 1000*9092*86.2176101*1.2=9.4e8

  @Now
  Scenario: 003, test market ETH/MAR52(sigma=10),

    Given the liquidity monitoring parameters:
      | name               | triggering ratio | time window | scaling factor |
      | updated-lqm-params | 0.1              | 24h         | 1              |
    When the markets are updated:
      | id        | liquidity monitoring | linear slippage factor | quadratic slippage factor |
      | ETH/MAR0  | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR11 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR12 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR21 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR22 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR23 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR31 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR32 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR41 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR42 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR43 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR51 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR52 | updated-lqm-params   | 1e6                    | 1e6                       |
      | ETH/MAR53 | updated-lqm-params   | 1e6                    | 1e6                       |
    And the following network parameters are set:
      | name                                  | value |
      | market.liquidity.bondPenaltyParameter | 0.2   |

    And the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR52 | 600000            | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR52 | 600000            | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference   |
      | party1 | ETH/MAR52 | buy  | 10     | 9     | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-11  |
      | party1 | ETH/MAR52 | buy  | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-12  |
      | party2 | ETH/MAR52 | sell | 1      | 10    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-13 |
      | party2 | ETH/MAR52 | sell | 10     | 11    | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-14 |

    When the opening auction period ends for market "ETH/MAR52"
    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR52"
    # And the network moves ahead "1" blocks

    And the market data for the market "ETH/MAR52" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 10         | TRADING_MODE_CONTINUOUS | 43200   | 4         | 24        | 557872       | 600000         | 1             |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1 x 10 x 1 x 55787.28815617000 =557872

    Then the order book should have the following volumes for market "ETH/MAR52":
      | side | price | volume |
      | sell | 20    | 30000  |
      | sell | 11    | 10     |
      | buy  | 9     | 10     |
      | buy  | 1     | 600000 |
    And the parties should have the following account balances:
      | party  | asset | market id | margin      | general        | bond   |
      | party0 | USD   | ETH/MAR52 | 20083423736 | 49979915976264 | 600000 |
      | party1 | USD   | ETH/MAR52 | 133         | 49999999999867 |        |
      | party2 | USD   | ETH/MAR52 | 8033370     | 49999991966630 |        |

# initial margin level for LP = 10*114559*55787.2881561700*1.2=7.66e10

