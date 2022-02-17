Feature: Replicate LP getting distressed during continuous trading, check if penalty is implemented correctly

  Background:
    Given the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
    And the average block duration is "1"
    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r   | sigma |
      | 0.000001      | 0.1 | 0  | 0  | 1.0    |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring updated every "1" seconds named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | oracle config          | maturity date        |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 2021-12-31T23:59:59Z |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount     |
      | party0 | USD   | 500000     |
      | party1 | USD   | 100000000  |
      | party2 | USD   | 100000000  |
      | party3 | USD   | 100000000  |
  

  Scenario: LP gets distressed during continuous trading

    Given the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee   | side | pegged reference | proportion | offset | lp type |
      | lp1 | party0 | ETH/MAR22 | 50000              | 0.001 | sell | ASK              | 500        | 20      | submission|
      | lp1 | party0 | ETH/MAR22 | 50000              | 0.001 | buy  | BID              | 500        | -20     | amendment |
      
    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 50000          | 10            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price    | volume |
      | sell | 1100     | 1      |
      | sell | 1020     | 0      |
      | sell | 1010     | 101    |
      | sell | 1000     | 0      |
      | buy  | 1000     | 0      |
      | buy  | 990      | 103    |
      | buy  | 980      | 0      |
      | buy  | 900      | 1      |
      
    # check the requried balances 
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond |
      | party0 | USD   | ETH/MAR22 | 426829 | 23171    | 50000|
      | party1 | USD   | ETH/MAR22 | 12190  | 99987810 |  0   |
      | party2 | USD   | ETH/MAR22 | 51879  | 99948121 |  0   |
    #check the margin levels   
    Then the parties should have the following margin levels: 
      | party  | market id | maintenance | search | initial | release  |
      | party0 | ETH/MAR22 | 355691      | 391260 | 426829  | 497967   |
      | party1 | ETH/MAR22 | 10159       | 11174  | 12190   | 14222    |
      | party2 | ETH/MAR22 | 43233       | 47556  | 51879   | 60526    |
    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 |-10     | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1   |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    # #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price    | volume |
      | sell | 1100     | 1      |
      | sell | 1010     | 101    |
      | buy  | 1000     | 130    |
      | buy  | 990      | 1      |
      | buy  | 900      | 1      |
    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party2-sell-4  |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 213414       | 50000          | 60            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1010 x 13 x 1 x 0.1
    # target stake 1313 with target trigger on 0.6 -> ~788 triggers liquidity auction

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond |
      | party0 | USD   | ETH/MAR22 | 426829 | 23251    | 50000|
      | party1 | USD   | ETH/MAR22 | 12190  | 99987810 |  0   |
      | party2 | USD   | ETH/MAR22 | 264754 | 99734946 |  0   |
    #check the margin levels   
    Then the parties should have the following margin levels: 
      | party  | market id | maintenance | search | initial | release  |
      | party0 | ETH/MAR22 | 355691      | 391260 | 426829  | 497967   |

    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price    | volume |
      | sell | 1100     | 1      |
      | sell | 1010     | 101    |
      | sell | 1000     | 0      |
      | buy  | 1000     | 0      |
      | buy  | 990      | 103    |
      | buy  | 900      | 1      |

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond |
      | party0 | USD   | ETH/MAR22 | 426829 | 23251    | 50000|
      | party1 | USD   | ETH/MAR22 | 12190  | 99987810 |  0   |
      | party2 | USD   | ETH/MAR22 | 264754 | 99734946 |  0   |
      | party3 | USD   | ETH/MAR22 | 28826  | 99971294 |  0   |
      
   And the insurance pool balance should be "0" for the market "ETH/MAR22"

   When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference      |
      | party0 | ETH/MAR22 | sell | 15     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party0-sell-3  |

  Then debug transfers  
  Then debug orders

  And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 213414       | 50000          | 60            |

  And the insurance pool balance should be "8154" for the market "ETH/MAR22"
    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price    | volume |
      | sell | 1100     | 1      |
      | sell | 1010     | 1      |
      | sell | 1000     | 100    |
      | buy  | 1000     | 0      |
      | buy  | 990      | 103    |
      | buy  | 900      | 1      |

  #check the requried balances 
   And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond |
      | party0 | USD   | ETH/MAR22 | 490852 | 0        | 1074 |
      | party1 | USD   | ETH/MAR22 | 12190  | 99987810 |  0   |
      | party2 | USD   | ETH/MAR22 | 264754 | 99734946 |  0   |
      | party3 | USD   | ETH/MAR22 | 28826  | 99971294 |  0   |

   Then the parties should have the following margin levels: 
      | party  | market id | maintenance | search | initial | release  |
      | party0 | ETH/MAR22 | 355691      | 391260 | 426829  | 497967   |
      | party1 | ETH/MAR22 | 10159       | 11174  | 12190   | 14222    |
      | party2 | ETH/MAR22 | 221129      | 243241 | 265354  | 309580   |