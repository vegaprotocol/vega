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
    
    Given the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future |

    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |

   Scenario: 001, original risk model

   And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |

   And the average block duration is "1"

   And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0.001 | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 1688         | 50000          | 10            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf_short = 1000 x 10 x 1 x 0.16882368861315200 =1689

    #check the volume on the order book, liquidity price has been kept within price monitoring bounds 
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 92     |
      | sell | 1120  | 0      |
      | buy  | 900   | 113    |
      | buy  | 880   | 0      |

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general   | bond  |
      | party0 | USD   | ETH/MAR22 | 19524  | 430476   | 50000 |
      | party1 | USD   | ETH/MAR22 | 3117   | 99996883 | 0     |
      | party2 | USD   | ETH/MAR22 | 3429   | 99996571 | 0     |
      
    #check the margin levels
    #party1 margin level is: margin_position+margin_order = vol * (MarkPrice-ExitPrice)+ vol * rf * MarkPrice + order * rf * MarkPrice = 10 * (1000-900)+10*0.145263949*1000 + 1*0.145263949*1000=2598
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 16270       | 17897  | 19524  | 22778  |
      | party1 | ETH/MAR22 | 2598        | 2857   | 3117   | 3637   |
      | party2 | ETH/MAR22 | 2858        | 3143   | 3429   | 4001   |

    

  
