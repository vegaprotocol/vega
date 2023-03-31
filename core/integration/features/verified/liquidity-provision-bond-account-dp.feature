Feature: Check that bond slashing works with non-default asset decimals, market decimals, position decimals.

  Background:

    Given the log normal risk model named "log-normal-risk-model-1":
      | risk aversion | tau | mu | r | sigma |
      | 0.000001      | 0.1 | 0  | 0 | 1.0   |
    And the following assets are registered:
      | id  | decimal places |
      | USD | 3              |
    And the fees configuration named "fees-config-1":
      | maker fee | infrastructure fee |
      | 0.004     | 0.001              |
    And the price monitoring named "price-monitoring-1":
      | horizon | probability | auction extension |
      | 1       | 0.99        | 300               |
    And the following network parameters are set:
      | name                                          | value |
      | market.stake.target.timeWindow                | 24h   |
      | market.stake.target.scalingFactor             | 1     |
      | market.liquidity.bondPenaltyParameter         | 0.1   |
      | market.liquidity.targetstake.triggering.ratio | 0.24  |
    And the markets:
      | id        | quote name | asset | risk model              | margin calculator         | auction duration | fees          | price monitoring   | data source config     | decimal places | position decimal places | linear slippage factor | quadratic slippage factor |
      | ETH/MAR22 | ETH        | USD   | log-normal-risk-model-1 | default-margin-calculator | 1                | fees-config-1 | price-monitoring-1 | default-eth-for-future | 1              | 2                       | 0.7                    | 0                         |
    And the parties deposit on asset's general account the following amount:
      | party  | asset | amount    |
      | party0 | USD   | 500000    |
      | party1 | USD   | 100000000 |
      | party2 | USD   | 100000000 |
      | party3 | USD   | 100000000 |
      | party4 | USD   | 100000000 |
      | party5 | USD   | 100000000 |
    And the following network parameters are set:
      | name                                    | value |
      | network.markPriceUpdateMaximumFrequency | 0s    |

  @Now
  Scenario: Bond slashing on LP (0044-LIME-002, 0035-LIQM-004, 0044-LIME-009 )

    Given the average block duration is "1"

    And the parties submit the following liquidity provision:
      | id  | party  | market id | commitment amount | fee | side | pegged reference | proportion | offset | lp type    |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0   | sell | ASK              | 500        | 20     | submission |
      | lp1 | party0 | ETH/MAR22 | 50000             | 0   | buy  | BID              | 500        | 20     | amendment  |

    And the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference  |
      | party4 | ETH/MAR22 | buy  | 100    | 850   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-4  |
      | party1 | ETH/MAR22 | buy  | 1      | 900   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 1      | 990   | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-1  |
      | party1 | ETH/MAR22 | buy  | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | buy-ref-2  |
      | party2 | ETH/MAR22 | sell | 10     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-3 |
      | party2 | ETH/MAR22 | sell | 1      | 1010  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-1 |
      | party2 | ETH/MAR22 | sell | 1      | 1100  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-2 |
      | party5 | ETH/MAR22 | sell | 100    | 1200  | 0                | TYPE_LIMIT | TIF_GTC | sell-ref-5 |


    When the opening auction period ends for market "ETH/MAR22"
    Then the auction ends with a traded volume of "10" at a price of "1000"
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = (100 x 0.1 x 1 x 3.5569)x 1000=35569
    And the insurance pool balance should be "0" for the market "ETH/MAR22"
    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 35569        | 50000          | 10            |

    #check the volume on the order book
    Then the order book should have the following volumes for market "ETH/MAR22":
      | side | price | volume |
      | sell | 1100  | 1      |
      | sell | 1030  | 49     |
      | sell | 1010  | 1      |
      | buy  | 990   | 1      |
      | buy  | 970   | 52     |
      | buy  | 900   | 1      |

    # check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 240854   | 50000 |
      | party1 | USD   | ETH/MAR22 | 11425  | 99988575 |       |
      | party2 | USD   | ETH/MAR22 | 51690  | 99948310 |       |
    #check the margin levels
    Then the parties should have the following margin levels:
      | party  | market id | maintenance | initial |
      | party0 | ETH/MAR22 | 174289      | 209146  |
      | party1 | ETH/MAR22 | 9889        | 11866   |
      | party2 | ETH/MAR22 | 42963       | 51555   |
    #check position (party0 has no position)
    Then the parties should have the following profit and loss:
      | party  | volume | unrealised pnl | realised pnl |
      | party1 | 10     | 0              | 0            |
      | party2 | -10    | 0              | 0            |

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference    |
      | party3 | ETH/MAR22 | buy  | 30     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party3-buy-1 |

    And the trading mode should be "TRADING_MODE_CONTINUOUS" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party2 | ETH/MAR22 | sell | 50     | 1000  | 1                | TYPE_LIMIT | TIF_GTC | party2-sell-4 |

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 142276       | 50000          | 40            |
    # target_stake = mark_price x max_oi x target_stake_scaling_factor x rf = (100 x 0.6 x 1 x 3.5569)x 1000=213414

    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 209146 | 240854   | 50000 |
      | party1 | USD   | ETH/MAR22 | 11425  | 99988575 |       |
      | party2 | USD   | ETH/MAR22 | 264970 | 99734880 |       |

    And the insurance pool balance should be "0" for the market "ETH/MAR22"

    When the parties place the following orders:
      | party  | market id | side | volume | price | resulting trades | type       | tif     | reference     |
      | party0 | ETH/MAR22 | sell | 60     | 1000  | 0                | TYPE_LIMIT | TIF_GTC | party0-sell-3 |
      | party1 | ETH/MAR22 | buy  | 100    | 1000  | 2                | TYPE_LIMIT | TIF_GTC | party1-buy-4  |

    # extra margin for party0: 15*1000*3.5569*1.2=64024
    # required margin: 64024+426829=490852
    # bond slashed amount: (490852-450000)*0.1=4085

    And the market data for the market "ETH/MAR22" should be:
      | mark price | trading mode            | horizon | min bound | max bound | target stake | supplied stake | open interest |
      | 1000       | TRADING_MODE_CONTINUOUS | 1       | 1000      | 1000      | 426828       | 50000          | 120           |

    And the insurance pool balance should be "1951" for the market "ETH/MAR22"

    #check the requried balances
    And the parties should have the following account balances:
      | party  | asset | market id | margin | general  | bond  |
      | party0 | USD   | ETH/MAR22 | 469299 | 453     | 28537 |
      | party1 | USD   | ETH/MAR22 | 107954 | 99891646 |       |
      | party2 | USD   | ETH/MAR22 | 264970 | 99734960 |       |
      | party3 | USD   | ETH/MAR22 | 28826  | 99971294 |       |

    Then the parties should have the following margin levels:
      | party  | market id | maintenance | search | initial | release |
      | party0 | ETH/MAR22 | 391083      | 430191 | 469299  | 547516  |
      | party1 | ETH/MAR22 | 89962       | 98958  | 107954  | 125946  |
      | party2 | ETH/MAR22 | 220809      | 242889 | 264970  | 309132  |

    # move to the next block to perform liquidity check
    Then the network moves ahead "1" blocks
    # open interest updates to include buy order of size 20
    And the market data for the market "ETH/MAR22" should be:
      | trading mode                    | auction trigger                          | target stake | supplied stake | open interest |
      | TRADING_MODE_MONITORING_AUCTION | AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET | 426828       | 50000          | 120           |

