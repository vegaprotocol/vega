Feature: check the impact from change of market parameter: market.liquidity.stakeToCcyVolume
  Background:
    Given time is updated to "2020-11-30T00:00:00Z"

    And the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the oracle spec for settlement data filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property         | type         | binding         |
      | prices.ETH.value | TYPE_INTEGER | settlement data |
    And the oracle spec for trading termination filtering data from "0xCAFECAFE1" named "ethDec21Oracle":
      | property           | type         | binding             |
      | trading.terminated | TYPE_BOOLEAN | trading termination |

    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0         | 0                  |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1000    | 0.99        | 300               |
    And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.2   |
      | market.liquidity.targetstake.triggering.ratio | 0.1   |
      | network.markPriceUpdateMaximumFrequency       | 0s    |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config | lp price range | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | ethDec21Oracle     | 0.014          | 1e6                    | 1e6                       |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000000 |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
    And the average block duration is "1"
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |

  Scenario: 001, market.liquidity.stakeToCcyVolume=2, 0007-POSN-010, 0013-ACCT-020
    Given the following network parameters are set:
      | name                              | value |
      | market.liquidity.stakeToCcyVolume | 2     |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | buy  | BID              | 500        | 20     | amendment  |

    # party1 :=  buy order volume * vwap * rf_long  = (900  + 990  + 50 * 1000) * 0.8007282079844139 =  41549.786712311237271
    # party2 := sell order volume * vwap * rf_short = (1100 + 1010 + 50 * 1000) * 3.556903591579342  = 185350.246157199511620
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial  |
      | party1 | ETH/MAR22 | 41550       | 49860    |
      | party2 | ETH/MAR22 | 185351      | 222421   |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 177845       | 5000000        | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1014  | 9862   |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 986   | 10142  |
      | buy  | 900   | 1      |

    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 50     | 0              | 0            |
      | party2 | -50    | 0              | 0            |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial  |
      | party0 | ETH/MAR22 | 35078184    | 42093820 |
      | party1 | ETH/MAR22 | 42338       | 50805    |
      | party2 | ETH/MAR22 | 185609      | 222730   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin   | general   |
      | party0 | USD   | ETH/MAR22 | 42093820 | 452906180 |
      | party1 | USD   | ETH/MAR22 | 49860    | 99950140  |
      | party2 | USD   | ETH/MAR22 | 222421   | 99777579  |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 2      | 1014  | 2                | TYPE_LIMIT | TIF_GTC | buy-p1-2  |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -1     | 0              | 0            |
      | party1 | 52     | 704            | 0            |
      | party2 | -51    | -704           | 0            |

  Scenario: 002, market.liquidity.stakeToCcyVolume=0.5,
    Given the following network parameters are set:
      | name                              | value |
      | market.liquidity.stakeToCcyVolume | 0.5   |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | buy  | BID              | 500        | 20     | amendment  |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 177845       | 5000000        | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1014  | 2466   |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 986   | 2536   |
      | buy  | 900   | 1      |

    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 50     | 0              | 0            |
      | party2 | -50    | 0              | 0            |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial  |
      | party0 | ETH/MAR22 | 8771325     | 10525590 |
      | party1 | ETH/MAR22 | 42338       | 50805    |
      | party2 | ETH/MAR22 | 185609      | 222730   |

    Then the parties should have the following account balances:
      | party  | asset | market id | margin   | general   |
      | party0 | USD   | ETH/MAR22 | 10525590 | 484474410 |
      | party1 | USD   | ETH/MAR22 | 49860    | 99950140  |
      | party2 | USD   | ETH/MAR22 | 222421   | 99777579  |

    When the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference |
      | party1 | ETH/MAR22 | buy  | 2      | 1014  | 2                | TYPE_LIMIT | TIF_GTC | buy-p1-2  |

    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party0 | -1     | 0              | 0            |
      | party1 | 52     | 704            | 0            |
      | party2 | -51    | -704           | 0            |

  Scenario: 003, market.liquidity.stakeToCcyVolume=0
    Given the following network parameters are set:
      | name                              | value |
      | market.liquidity.stakeToCcyVolume | 0     |
    And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10000  | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10000  | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 5000000           | 0   | buy  | BID              | 500        | 20     | amendment  |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 177845       | 5000000        | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 10001  |
      | sell | 1014  | 0      |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 986   | 0      |
      | buy  | 900   | 10001  |


  Scenario: 004, market.liquidity.stakeToCcyVolume=0, 3 LPs make commitment, 0044-LIME-012
    Given the parties deposit on asset's general account the following amount:
      | party   | asset | amount    |
      | party00 | USD   | 500000000 |
      | party01 | USD   | 500000000 |
      | party02 | USD   | 500000000 |
    And the following network parameters are set:
      | name                              | value |
      | market.liquidity.stakeToCcyVolume | 0     |
     And the parties place the following orders with ticks:
      | party  | market id | side | volume | price | resulting trades | type       | tif     |
      | party1 | ETH/MAR22 | buy  | 10000  | 900   | 0                | TYPE_LIMIT | TIF_GTC |
      | party2 | ETH/MAR22 | sell | 10000  | 1100  | 0                | TYPE_LIMIT | TIF_GTC |
    And the parties submit the following liquidity provision:
      | id  | party   | market id | commitment amount | fee  | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party00 | ETH/MAR22 | 17784             | 0.01 | sell | ASK              | 500        | 20     | submission |
      | lp1 | party00 | ETH/MAR22 | 17784             | 0.01 | buy  | BID              | 500        | 20     | amendment  |
      | lp2 | party01 | ETH/MAR22 | 177845            | 0.02 | sell | ASK              | 500        | 20     | submission |
      | lp2 | party01 | ETH/MAR22 | 177845            | 0.02 | buy  | BID              | 500        | 20     | amendment  |
      | lp3 | party02 | ETH/MAR22 | 27784             | 0.03 | sell | ASK              | 500        | 20     | submission |
      | lp3 | party02 | ETH/MAR22 | 27784             | 0.03 | buy  | BID              | 500        | 20     | amendment  |

    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "50" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = 1000 x 10 x 1 x 0.1
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1000    | 986       | 1014      | 177845       | 223413         | 50            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 10001  |
      | sell | 1014  | 0      |
      | sell | 1010  | 1      |
      | sell | 1000  | 0      |
      | buy  | 1000  | 0      |
      | buy  | 990   | 1      |
      | buy  | 986   | 0      |
      | buy  | 900   | 10001  |    
    And the liquidity fee factor should be "0.02" for the market "ETH/MAR22"
